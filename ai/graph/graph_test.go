// Package graph - tests for P3-C001.
package graph

import (
	"testing"
	"time"
)

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple title",
			content:  "Hello World",
			expected: "Hello World",
		},
		{
			name:     "markdown h1",
			content:  "# Hello World",
			expected: "Hello World",
		},
		{
			name:     "markdown h2",
			content:  "## Hello World",
			expected: "Hello World",
		},
		{
			name:     "markdown h3",
			content:  "### Hello World",
			expected: "Hello World",
		},
		{
			name:     "multiline takes first",
			content:  "First Line\nSecond Line",
			expected: "First Line",
		},
		{
			name:     "truncates long titles",
			content:  "This is a very long title that exceeds fifty characters and should be truncated",
			expected: "This is a very long title that exceeds fifty chara...",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name:     "whitespace handling",
			content:  "  # Spaced Title  ",
			expected: "Spaced Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.content)
			if result != tt.expected {
				t.Errorf("extractTitle(%q) = %q, want %q", tt.content, result, tt.expected)
			}
		})
	}
}

func TestComputePageRank(t *testing.T) {
	tests := []struct {
		graph         *KnowledgeGraph
		name          string
		checkHighest  string
		expectNonZero bool
	}{
		{
			name: "empty graph",
			graph: &KnowledgeGraph{
				Nodes: []GraphNode{},
				Edges: []GraphEdge{},
			},
			expectNonZero: false,
		},
		{
			name: "single node",
			graph: &KnowledgeGraph{
				Nodes: []GraphNode{
					{ID: "A", Label: "Node A"},
				},
				Edges: []GraphEdge{},
			},
			expectNonZero: true,
		},
		{
			name: "star topology - center should be highest",
			graph: &KnowledgeGraph{
				Nodes: []GraphNode{
					{ID: "center", Label: "Center"},
					{ID: "leaf1", Label: "Leaf 1"},
					{ID: "leaf2", Label: "Leaf 2"},
					{ID: "leaf3", Label: "Leaf 3"},
				},
				Edges: []GraphEdge{
					{Source: "leaf1", Target: "center", Type: EdgeTypeLink, Weight: 1.0},
					{Source: "leaf2", Target: "center", Type: EdgeTypeLink, Weight: 1.0},
					{Source: "leaf3", Target: "center", Type: EdgeTypeLink, Weight: 1.0},
				},
			},
			expectNonZero: true,
			checkHighest:  "center",
		},
		{
			name: "linear chain",
			graph: &KnowledgeGraph{
				Nodes: []GraphNode{
					{ID: "A", Label: "A"},
					{ID: "B", Label: "B"},
					{ID: "C", Label: "C"},
				},
				Edges: []GraphEdge{
					{Source: "A", Target: "B", Type: EdgeTypeLink, Weight: 1.0},
					{Source: "B", Target: "C", Type: EdgeTypeLink, Weight: 1.0},
				},
			},
			expectNonZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &GraphBuilder{config: DefaultConfig()}
			b.computePageRank(tt.graph)

			if tt.expectNonZero && len(tt.graph.Nodes) > 0 {
				hasNonZero := false
				for _, node := range tt.graph.Nodes {
					if node.Importance > 0 {
						hasNonZero = true
						break
					}
				}
				if !hasNonZero {
					t.Error("expected non-zero importance scores")
				}
			}

			if tt.checkHighest != "" {
				var maxScore float64
				var maxID string
				for _, node := range tt.graph.Nodes {
					if node.Importance > maxScore {
						maxScore = node.Importance
						maxID = node.ID
					}
				}
				if maxID != tt.checkHighest {
					t.Errorf("expected %s to have highest score, got %s", tt.checkHighest, maxID)
				}
			}
		})
	}
}

func TestDetectCommunities(t *testing.T) {
	tests := []struct {
		graph               *KnowledgeGraph
		name                string
		expectedMinClusters int
		expectedMaxClusters int
	}{
		{
			name: "empty graph",
			graph: &KnowledgeGraph{
				Nodes: []GraphNode{},
				Edges: []GraphEdge{},
			},
			expectedMinClusters: 0,
			expectedMaxClusters: 0,
		},
		{
			name: "disconnected nodes - each its own cluster",
			graph: &KnowledgeGraph{
				Nodes: []GraphNode{
					{ID: "A", Label: "A"},
					{ID: "B", Label: "B"},
					{ID: "C", Label: "C"},
				},
				Edges: []GraphEdge{},
			},
			expectedMinClusters: 3,
			expectedMaxClusters: 3,
		},
		{
			name: "fully connected - should merge",
			graph: &KnowledgeGraph{
				Nodes: []GraphNode{
					{ID: "A", Label: "A"},
					{ID: "B", Label: "B"},
					{ID: "C", Label: "C"},
				},
				Edges: []GraphEdge{
					{Source: "A", Target: "B", Type: EdgeTypeLink, Weight: 1.0},
					{Source: "B", Target: "C", Type: EdgeTypeLink, Weight: 1.0},
					{Source: "A", Target: "C", Type: EdgeTypeLink, Weight: 1.0},
				},
			},
			expectedMinClusters: 1,
			expectedMaxClusters: 1,
		},
		{
			name: "two separate clusters",
			graph: &KnowledgeGraph{
				Nodes: []GraphNode{
					{ID: "A1", Label: "A1"},
					{ID: "A2", Label: "A2"},
					{ID: "B1", Label: "B1"},
					{ID: "B2", Label: "B2"},
				},
				Edges: []GraphEdge{
					{Source: "A1", Target: "A2", Type: EdgeTypeLink, Weight: 1.0},
					{Source: "B1", Target: "B2", Type: EdgeTypeLink, Weight: 1.0},
				},
			},
			expectedMinClusters: 2,
			expectedMaxClusters: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &GraphBuilder{config: DefaultConfig()}
			count := b.detectCommunities(tt.graph)

			if count < tt.expectedMinClusters || count > tt.expectedMaxClusters {
				t.Errorf("detectCommunities() = %d, want between %d and %d",
					count, tt.expectedMinClusters, tt.expectedMaxClusters)
			}
		})
	}
}

func TestApplyFilter(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	baseGraph := &KnowledgeGraph{
		Nodes: []GraphNode{
			{ID: "1", Label: "Go Programming", Tags: []string{"go", "programming"}, Importance: 0.9, Cluster: 0, CreatedAt: now},
			{ID: "2", Label: "Python Tips", Tags: []string{"python", "programming"}, Importance: 0.7, Cluster: 0, CreatedAt: yesterday},
			{ID: "3", Label: "Cooking Recipe", Tags: []string{"cooking"}, Importance: 0.3, Cluster: 1, CreatedAt: lastWeek},
			{ID: "4", Label: "Travel Plans", Tags: []string{"travel"}, Importance: 0.5, Cluster: 2, CreatedAt: now},
		},
		Edges: []GraphEdge{
			{Source: "1", Target: "2", Type: EdgeTypeTagCo, Weight: 0.5},
			{Source: "1", Target: "4", Type: EdgeTypeSemantic, Weight: 0.8},
		},
	}

	tests := []struct {
		filter        GraphFilter
		name          string
		expectedNodes int
		expectedEdges int
	}{
		{
			name:          "no filter",
			filter:        GraphFilter{},
			expectedNodes: 4,
			expectedEdges: 2,
		},
		{
			name: "filter by tag",
			filter: GraphFilter{
				Tags: []string{"programming"},
			},
			expectedNodes: 2,
			expectedEdges: 1,
		},
		{
			name: "filter by importance",
			filter: GraphFilter{
				MinImportance: 0.6,
			},
			expectedNodes: 2,
			expectedEdges: 1,
		},
		{
			name: "filter by cluster",
			filter: GraphFilter{
				Clusters: []int{0},
			},
			expectedNodes: 2,
			expectedEdges: 1,
		},
		{
			name: "filter by multiple clusters",
			filter: GraphFilter{
				Clusters: []int{1, 2},
			},
			expectedNodes: 2,
			expectedEdges: 0,
		},
		{
			name: "combined filters",
			filter: GraphFilter{
				Tags:          []string{"programming"},
				MinImportance: 0.8,
			},
			expectedNodes: 1,
			expectedEdges: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyFilter(baseGraph, tt.filter)

			if len(result.Nodes) != tt.expectedNodes {
				t.Errorf("ApplyFilter() nodes = %d, want %d", len(result.Nodes), tt.expectedNodes)
			}
			if len(result.Edges) != tt.expectedEdges {
				t.Errorf("ApplyFilter() edges = %d, want %d", len(result.Edges), tt.expectedEdges)
			}
		})
	}
}

func TestApplyFilterNil(t *testing.T) {
	result := ApplyFilter(nil, GraphFilter{})
	if result != nil {
		t.Error("ApplyFilter(nil) should return nil")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MinTagSimilarity != 0.1 {
		t.Errorf("MinTagSimilarity = %f, want 0.1", config.MinTagSimilarity)
	}
	if config.MinSemanticSimilarity != 0.7 {
		t.Errorf("MinSemanticSimilarity = %f, want 0.7", config.MinSemanticSimilarity)
	}
	if config.MaxSemanticEdgesPerNode != 3 {
		t.Errorf("MaxSemanticEdgesPerNode = %d, want 3", config.MaxSemanticEdgesPerNode)
	}
	if !config.EnableCommunityDetection {
		t.Error("EnableCommunityDetection should be true")
	}
	if !config.EnablePageRank {
		t.Error("EnablePageRank should be true")
	}
}

func TestGraphConstants(t *testing.T) {
	// Edge types
	if EdgeTypeLink != "link" {
		t.Errorf("EdgeTypeLink = %s, want link", EdgeTypeLink)
	}
	if EdgeTypeTagCo != "tag_co" {
		t.Errorf("EdgeTypeTagCo = %s, want tag_co", EdgeTypeTagCo)
	}
	if EdgeTypeSemantic != "semantic" {
		t.Errorf("EdgeTypeSemantic = %s, want semantic", EdgeTypeSemantic)
	}

	// Node types
	if NodeTypeMemo != "memo" {
		t.Errorf("NodeTypeMemo = %s, want memo", NodeTypeMemo)
	}
	if NodeTypeTag != "tag" {
		t.Errorf("NodeTypeTag = %s, want tag", NodeTypeTag)
	}
}
