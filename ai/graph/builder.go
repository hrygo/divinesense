// Package graph - builder implementation for P3-C001.
package graph

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/duplicate"
	"github.com/hrygo/divinesense/store"
)

// GraphBuilder builds knowledge graphs from memos.
type GraphBuilder struct {
	store     *store.Store
	embedding ai.EmbeddingService
	model     string
	config    GraphConfig
}

// NewGraphBuilder creates a new GraphBuilder.
func NewGraphBuilder(s *store.Store, embedding ai.EmbeddingService, model string) *GraphBuilder {
	return &GraphBuilder{
		store:     s,
		embedding: embedding,
		model:     model,
		config:    DefaultConfig(),
	}
}

// NewGraphBuilderWithConfig creates a builder with custom config.
func NewGraphBuilderWithConfig(s *store.Store, embedding ai.EmbeddingService, model string, config GraphConfig) *GraphBuilder {
	return &GraphBuilder{
		store:     s,
		embedding: embedding,
		model:     model,
		config:    config,
	}
}

// Build constructs the knowledge graph for a user.
func (b *GraphBuilder) Build(ctx context.Context, userID int32) (*KnowledgeGraph, error) {
	start := time.Now()
	graph := &KnowledgeGraph{}

	// Step 1: Get all memos as nodes
	memos, err := b.store.ListMemos(ctx, &store.FindMemo{
		CreatorID: &userID,
	})
	if err != nil {
		return nil, fmt.Errorf("list memos: %w", err)
	}

	if len(memos) == 0 {
		graph.BuildMs = time.Since(start).Milliseconds()
		return graph, nil
	}

	// Build node map and ID->UID map for quick lookup (avoid N+1 queries)
	nodeMap := make(map[string]*GraphNode)
	memoIDToUID := make(map[int32]string)
	for _, memo := range memos {
		node := &GraphNode{
			ID:        memo.UID,
			Label:     extractTitle(memo.Content),
			Type:      NodeTypeMemo,
			Tags:      extractTags(memo),
			CreatedAt: time.Unix(memo.CreatedTs, 0),
		}
		graph.Nodes = append(graph.Nodes, *node)
		nodeMap[memo.UID] = node
		memoIDToUID[memo.ID] = memo.UID
	}

	// Step 2: Build edges
	// 2.1 Explicit links (from memo relations)
	linkEdges := b.buildLinkEdges(ctx, memoIDToUID, nodeMap)
	graph.Edges = append(graph.Edges, linkEdges...)
	graph.Stats.LinkEdges = len(linkEdges)

	// 2.2 Tag co-occurrence
	tagEdges := b.buildTagCoOccurrenceEdges(memos, nodeMap)
	graph.Edges = append(graph.Edges, tagEdges...)
	graph.Stats.TagEdges = len(tagEdges)

	// 2.3 Semantic similarity
	semanticEdges := b.buildSemanticEdges(ctx, memos, nodeMap)
	graph.Edges = append(graph.Edges, semanticEdges...)
	graph.Stats.SemanticEdges = len(semanticEdges)

	// Step 3: Compute importance (PageRank)
	if b.config.EnablePageRank {
		b.computePageRank(graph)
	}

	// Step 4: Community detection
	if b.config.EnableCommunityDetection {
		clusterCount := b.detectCommunities(graph)
		graph.Stats.ClusterCount = clusterCount
	}

	// Update stats
	graph.Stats.NodeCount = len(graph.Nodes)
	graph.Stats.EdgeCount = len(graph.Edges)
	graph.BuildMs = time.Since(start).Milliseconds()

	return graph, nil
}

// buildLinkEdges creates edges from explicit memo relations.
// memoIDToUID is pre-computed to avoid N+1 queries.
func (b *GraphBuilder) buildLinkEdges(ctx context.Context, memoIDToUID map[int32]string, nodeMap map[string]*GraphNode) []GraphEdge {
	var edges []GraphEdge

	// Get memo relations - only for memos we care about (filtered by memoIDToUID)
	relations, err := b.store.ListMemoRelations(ctx, &store.FindMemoRelation{})
	if err != nil {
		slog.Warn("failed to list memo relations", "error", err)
		return edges
	}

	// Create edges from relations
	seen := make(map[string]bool)
	for _, rel := range relations {
		sourceUID := memoIDToUID[rel.MemoID]
		targetUID := memoIDToUID[rel.RelatedMemoID]

		// Skip if either node not in graph (filters to user's memos only)
		if sourceUID == "" || targetUID == "" {
			continue
		}

		// Deduplicate (only one edge per pair)
		edgeKey := fmt.Sprintf("%s-%s", sourceUID, targetUID)
		reverseKey := fmt.Sprintf("%s-%s", targetUID, sourceUID)
		if seen[edgeKey] || seen[reverseKey] {
			continue
		}
		seen[edgeKey] = true

		edges = append(edges, GraphEdge{
			Source: sourceUID,
			Target: targetUID,
			Type:   EdgeTypeLink,
			Weight: 1.0,
		})
	}

	return edges
}

// buildTagCoOccurrenceEdges creates edges based on shared tags.
func (b *GraphBuilder) buildTagCoOccurrenceEdges(memos []*store.Memo, nodeMap map[string]*GraphNode) []GraphEdge {
	var edges []GraphEdge

	// Build tag to memo mapping
	tagToMemos := make(map[string][]string)
	for _, memo := range memos {
		tags := extractTags(memo)
		for _, tag := range tags {
			tagLower := strings.ToLower(tag)
			tagToMemos[tagLower] = append(tagToMemos[tagLower], memo.UID)
		}
	}

	// Create edges for memos sharing tags
	seen := make(map[string]bool)
	for _, memoUIDs := range tagToMemos {
		if len(memoUIDs) < 2 {
			continue
		}

		// Create edges between all pairs
		for i := 0; i < len(memoUIDs); i++ {
			for j := i + 1; j < len(memoUIDs); j++ {
				source, target := memoUIDs[i], memoUIDs[j]

				// Deduplicate
				edgeKey := fmt.Sprintf("%s-%s", source, target)
				if seen[edgeKey] {
					continue
				}
				seen[edgeKey] = true

				// Calculate weight based on tag similarity
				node1, node2 := nodeMap[source], nodeMap[target]
				if node1 == nil || node2 == nil {
					continue
				}

				weight := duplicate.TagCoOccurrence(node1.Tags, node2.Tags)
				if weight >= b.config.MinTagSimilarity {
					edges = append(edges, GraphEdge{
						Source: source,
						Target: target,
						Type:   EdgeTypeTagCo,
						Weight: weight,
					})
				}
			}
		}
	}

	return edges
}

// buildSemanticEdges creates edges based on semantic similarity.
func (b *GraphBuilder) buildSemanticEdges(ctx context.Context, memos []*store.Memo, nodeMap map[string]*GraphNode) []GraphEdge {
	var edges []GraphEdge

	if b.embedding == nil {
		return edges
	}

	// For each memo, find top-K similar memos
	seen := make(map[string]bool)
	for _, memo := range memos {
		// Get embedding
		embedding, err := b.store.GetMemoEmbedding(ctx, memo.ID, b.model)
		if err != nil || embedding == nil || len(embedding.Embedding) == 0 {
			continue
		}

		// Vector search for similar memos
		results, err := b.store.VectorSearch(ctx, &store.VectorSearchOptions{
			UserID: memo.CreatorID,
			Vector: embedding.Embedding,
			Limit:  b.config.MaxSemanticEdgesPerNode + 1, // +1 to exclude self
		})
		if err != nil {
			continue
		}

		count := 0
		for _, result := range results {
			if result.Memo == nil || result.Memo.UID == memo.UID {
				continue // Skip self
			}

			if float64(result.Score) < b.config.MinSemanticSimilarity {
				continue
			}

			if count >= b.config.MaxSemanticEdgesPerNode {
				break
			}

			// Deduplicate
			edgeKey := fmt.Sprintf("%s-%s", memo.UID, result.Memo.UID)
			reverseKey := fmt.Sprintf("%s-%s", result.Memo.UID, memo.UID)
			if seen[edgeKey] || seen[reverseKey] {
				continue
			}
			seen[edgeKey] = true

			edges = append(edges, GraphEdge{
				Source: memo.UID,
				Target: result.Memo.UID,
				Type:   EdgeTypeSemantic,
				Weight: float64(result.Score),
			})
			count++
		}
	}

	return edges
}

// computePageRank computes importance scores using simplified PageRank.
func (b *GraphBuilder) computePageRank(graph *KnowledgeGraph) {
	if len(graph.Nodes) == 0 {
		return
	}

	const (
		damping    = 0.85
		iterations = 20
	)

	// Initialize scores
	n := len(graph.Nodes)
	scores := make(map[string]float64)
	for i := range graph.Nodes {
		scores[graph.Nodes[i].ID] = 1.0 / float64(n)
	}

	// Build adjacency list
	outLinks := make(map[string][]string)
	inLinks := make(map[string][]string)
	for _, edge := range graph.Edges {
		outLinks[edge.Source] = append(outLinks[edge.Source], edge.Target)
		inLinks[edge.Target] = append(inLinks[edge.Target], edge.Source)
	}

	// Iterate
	for iter := 0; iter < iterations; iter++ {
		newScores := make(map[string]float64)
		for id := range scores {
			sum := 0.0
			for _, inID := range inLinks[id] {
				outDegree := len(outLinks[inID])
				if outDegree > 0 {
					sum += scores[inID] / float64(outDegree)
				}
			}
			newScores[id] = (1-damping)/float64(n) + damping*sum
		}
		scores = newScores
	}

	// Normalize to 0-1
	var maxScore float64
	for _, score := range scores {
		if score > maxScore {
			maxScore = score
		}
	}

	if maxScore > 0 {
		for i := range graph.Nodes {
			graph.Nodes[i].Importance = scores[graph.Nodes[i].ID] / maxScore
		}
	}
}

// detectCommunities performs simple community detection.
// Uses label propagation for simplicity.
func (b *GraphBuilder) detectCommunities(graph *KnowledgeGraph) int {
	if len(graph.Nodes) == 0 {
		return 0
	}

	// Initialize each node with its own community
	labels := make(map[string]int)
	for i, node := range graph.Nodes {
		labels[node.ID] = i
	}

	// Build adjacency
	neighbors := make(map[string][]string)
	for _, edge := range graph.Edges {
		neighbors[edge.Source] = append(neighbors[edge.Source], edge.Target)
		neighbors[edge.Target] = append(neighbors[edge.Target], edge.Source)
	}

	// Iterate label propagation
	const maxIterations = 10
	for iter := 0; iter < maxIterations; iter++ {
		changed := false
		for _, node := range graph.Nodes {
			if len(neighbors[node.ID]) == 0 {
				continue
			}

			// Count neighbor labels
			labelCount := make(map[int]int)
			for _, neighbor := range neighbors[node.ID] {
				labelCount[labels[neighbor]]++
			}

			// Find most common label
			maxCount := 0
			maxLabel := labels[node.ID]
			for label, count := range labelCount {
				if count > maxCount {
					maxCount = count
					maxLabel = label
				}
			}

			if labels[node.ID] != maxLabel {
				labels[node.ID] = maxLabel
				changed = true
			}
		}

		if !changed {
			break
		}
	}

	// Assign cluster IDs
	clusterMap := make(map[int]int)
	nextCluster := 0
	for i := range graph.Nodes {
		label := labels[graph.Nodes[i].ID]
		if _, ok := clusterMap[label]; !ok {
			clusterMap[label] = nextCluster
			nextCluster++
		}
		graph.Nodes[i].Cluster = clusterMap[label]
	}

	return nextCluster
}

// extractTitle extracts title from memo content (rune-safe for CJK).
func extractTitle(content string) string {
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) == 0 {
		return ""
	}
	title := strings.TrimSpace(lines[0])
	for strings.HasPrefix(title, "#") {
		title = strings.TrimPrefix(title, "#")
	}
	title = strings.TrimSpace(title)
	// Use rune slice for safe truncation (avoid cutting UTF-8 in the middle)
	runes := []rune(title)
	if len(runes) > 50 {
		return string(runes[:50]) + "..."
	}
	return title
}

// extractTags extracts tags from memo.
func extractTags(memo *store.Memo) []string {
	if memo == nil || memo.Payload == nil {
		return nil
	}
	return memo.Payload.Tags
}

// GetGraph retrieves a cached graph or builds a new one.
func (b *GraphBuilder) GetGraph(ctx context.Context, userID int32) (*KnowledgeGraph, error) {
	// For now, always build fresh
	// TODO: Add caching with cache service
	return b.Build(ctx, userID)
}

// GetFilteredGraph returns a filtered view of the graph.
func (b *GraphBuilder) GetFilteredGraph(ctx context.Context, userID int32, filter GraphFilter) (*KnowledgeGraph, error) {
	graph, err := b.GetGraph(ctx, userID)
	if err != nil {
		return nil, err
	}

	return ApplyFilter(graph, filter), nil
}

// GraphFilter contains filter criteria for graph visualization.
type GraphFilter struct {
	StartDate     *time.Time
	EndDate       *time.Time
	Tags          []string
	Clusters      []int
	MinImportance float64
}

// ApplyFilter filters the graph based on criteria.
func ApplyFilter(graph *KnowledgeGraph, filter GraphFilter) *KnowledgeGraph {
	if graph == nil {
		return nil
	}

	// Filter nodes
	nodeSet := make(map[string]bool)
	var filteredNodes []GraphNode

	for _, node := range graph.Nodes {
		// Tag filter
		if len(filter.Tags) > 0 {
			hasTag := false
			for _, filterTag := range filter.Tags {
				for _, nodeTag := range node.Tags {
					if strings.EqualFold(filterTag, nodeTag) {
						hasTag = true
						break
					}
				}
				if hasTag {
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Importance filter
		if node.Importance < filter.MinImportance {
			continue
		}

		// Cluster filter
		if len(filter.Clusters) > 0 {
			found := false
			for _, c := range filter.Clusters {
				if node.Cluster == c {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Date filter
		if filter.StartDate != nil && node.CreatedAt.Before(*filter.StartDate) {
			continue
		}
		if filter.EndDate != nil && node.CreatedAt.After(*filter.EndDate) {
			continue
		}

		filteredNodes = append(filteredNodes, node)
		nodeSet[node.ID] = true
	}

	// Filter edges (only keep edges where both nodes are included)
	var filteredEdges []GraphEdge
	var linkEdges, tagEdges, semanticEdges int
	for _, edge := range graph.Edges {
		if nodeSet[edge.Source] && nodeSet[edge.Target] {
			filteredEdges = append(filteredEdges, edge)
			switch edge.Type {
			case EdgeTypeLink:
				linkEdges++
			case EdgeTypeTagCo:
				tagEdges++
			case EdgeTypeSemantic:
				semanticEdges++
			}
		}
	}

	// Count unique clusters in filtered nodes
	clusterSet := make(map[int]bool)
	for _, node := range filteredNodes {
		clusterSet[node.Cluster] = true
	}

	// Sort nodes by importance
	sort.Slice(filteredNodes, func(i, j int) bool {
		return filteredNodes[i].Importance > filteredNodes[j].Importance
	})

	return &KnowledgeGraph{
		Nodes:   filteredNodes,
		Edges:   filteredEdges,
		BuildMs: graph.BuildMs,
		Stats: GraphStats{
			NodeCount:     len(filteredNodes),
			EdgeCount:     len(filteredEdges),
			ClusterCount:  len(clusterSet),
			LinkEdges:     linkEdges,
			TagEdges:      tagEdges,
			SemanticEdges: semanticEdges,
		},
	}
}
