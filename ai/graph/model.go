// Package graph provides knowledge graph construction for P3-C001.
package graph

import (
	"time"
)

// GraphNode represents a node in the knowledge graph.
type GraphNode struct {
	CreatedAt  time.Time `json:"created_at"`
	ID         string    `json:"id"`
	Label      string    `json:"label"`
	Type       string    `json:"type"`
	Tags       []string  `json:"tags,omitempty"`
	Importance float64   `json:"importance"`
	Cluster    int       `json:"cluster"`
	X          float64   `json:"x,omitempty"`
	Y          float64   `json:"y,omitempty"`
}

// GraphEdge represents an edge in the knowledge graph.
type GraphEdge struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Type   string  `json:"type"`   // "link", "tag_co", "semantic"
	Weight float64 `json:"weight"` // 0-1, higher = stronger connection
}

// KnowledgeGraph represents the complete graph structure.
type KnowledgeGraph struct {
	Nodes   []GraphNode `json:"nodes"`
	Edges   []GraphEdge `json:"edges"`
	Stats   GraphStats  `json:"stats"`
	BuildMs int64       `json:"build_ms"` // build latency
}

// GraphStats contains graph statistics.
type GraphStats struct {
	NodeCount     int `json:"node_count"`
	EdgeCount     int `json:"edge_count"`
	ClusterCount  int `json:"cluster_count"`
	LinkEdges     int `json:"link_edges"`
	TagEdges      int `json:"tag_edges"`
	SemanticEdges int `json:"semantic_edges"`
}

// EdgeType constants.
const (
	EdgeTypeLink     = "link"     // explicit user-created link
	EdgeTypeTagCo    = "tag_co"   // tag co-occurrence
	EdgeTypeSemantic = "semantic" // semantic similarity
)

// NodeType constants.
const (
	NodeTypeMemo = "memo"
	NodeTypeTag  = "tag"
)

// GraphConfig contains configuration for graph building.
type GraphConfig struct {
	// MinTagSimilarity is the minimum Jaccard similarity between tag sets to create an edge.
	MinTagSimilarity float64
	// MinSemanticSimilarity is the minimum similarity score for semantic edges.
	MinSemanticSimilarity float64
	// MaxSemanticEdgesPerNode limits semantic edges per node.
	MaxSemanticEdgesPerNode int
	// EnableCommunityDetection enables Louvain community detection.
	EnableCommunityDetection bool
	// EnablePageRank enables PageRank importance calculation.
	EnablePageRank bool
}

// DefaultConfig returns default graph configuration.
func DefaultConfig() GraphConfig {
	return GraphConfig{
		MinTagSimilarity:         0.1,
		MinSemanticSimilarity:    0.7,
		MaxSemanticEdgesPerNode:  3,
		EnableCommunityDetection: true,
		EnablePageRank:           true,
	}
}
