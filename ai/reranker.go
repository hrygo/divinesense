package ai

import (
	"github.com/hrygo/divinesense/ai/core/reranker"
)

// RerankResult represents a reranking result.
// Deprecated: Use reranker.Result directly.
type RerankResult = reranker.Result

// RerankerService is the reranking service interface.
// Deprecated: Use reranker.Service directly.
type RerankerService = reranker.Service

// NewRerankerService creates a new RerankerService.
//
// Phase 1 Note: This is a bridge compatibility layer that maintains the original API.
// The actual reranker functionality has been moved to ai/core/reranker/service.go.
func NewRerankerService(cfg *RerankerConfig) RerankerService {
	return reranker.NewService((*reranker.Config)(cfg))
}
