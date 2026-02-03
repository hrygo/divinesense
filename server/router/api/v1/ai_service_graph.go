// Package v1 - knowledge graph handlers for P3-C001.
package v1

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hrygo/divinesense/ai/graph"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
)

// GetKnowledgeGraph returns the knowledge graph for the current user.
func (s *AIService) GetKnowledgeGraph(ctx context.Context, req *v1pb.GetKnowledgeGraphRequest) (*v1pb.GetKnowledgeGraphResponse, error) {
	if !s.IsEnabled() {
		return nil, status.Errorf(codes.Unavailable, "AI features are disabled")
	}

	// Get current user
	user, err := getCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unauthorized")
	}

	// Create graph builder
	builder := graph.NewGraphBuilder(s.Store, s.EmbeddingService, s.EmbeddingModel)

	// Build filter
	filter := graph.GraphFilter{
		Tags:          req.Tags,
		MinImportance: req.MinImportance,
	}
	for _, c := range req.Clusters {
		filter.Clusters = append(filter.Clusters, int(c))
	}

	// Get filtered graph
	kg, err := builder.GetFilteredGraph(ctx, user.ID, filter)
	if err != nil {
		slog.Error("failed to build knowledge graph", "user_id", user.ID, "error", err)
		return nil, status.Error(codes.Internal, "failed to build graph")
	}

	// Convert to response
	response := &v1pb.GetKnowledgeGraphResponse{
		BuildMs: kg.BuildMs,
		Stats: &v1pb.GraphStats{
			NodeCount:     int32(kg.Stats.NodeCount),
			EdgeCount:     int32(kg.Stats.EdgeCount),
			ClusterCount:  int32(kg.Stats.ClusterCount),
			LinkEdges:     int32(kg.Stats.LinkEdges),
			TagEdges:      int32(kg.Stats.TagEdges),
			SemanticEdges: int32(kg.Stats.SemanticEdges),
		},
	}

	for _, node := range kg.Nodes {
		response.Nodes = append(response.Nodes, &v1pb.GraphNode{
			Id:         node.ID,
			Label:      node.Label,
			Type:       node.Type,
			Tags:       node.Tags,
			Importance: node.Importance,
			Cluster:    int32(node.Cluster),
			CreatedTs:  node.CreatedAt.Unix(),
		})
	}

	for _, edge := range kg.Edges {
		response.Edges = append(response.Edges, &v1pb.GraphEdge{
			Source: edge.Source,
			Target: edge.Target,
			Type:   edge.Type,
			Weight: edge.Weight,
		})
	}

	return response, nil
}
