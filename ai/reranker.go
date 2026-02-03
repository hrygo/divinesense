package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// RerankResult represents a reranking result.
type RerankResult struct {
	Index int     // Original index
	Score float32 // Relevance score
}

// RerankerService is the reranking service interface.
type RerankerService interface {
	// Rerank reorders documents by relevance.
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)

	// IsEnabled returns whether the service is enabled.
	IsEnabled() bool
}

type rerankerService struct {
	client  *http.Client
	apiKey  string
	baseURL string
	model   string
	enabled bool
}

// NewRerankerService creates a new RerankerService.
func NewRerankerService(cfg *RerankerConfig) RerankerService {
	return &rerankerService{
		enabled: cfg.Enabled,
		apiKey:  cfg.APIKey,
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

func (s *rerankerService) IsEnabled() bool {
	return s.enabled
}

func (s *rerankerService) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
	if !s.enabled {
		// Return original order when disabled
		results := make([]RerankResult, len(documents))
		for i := range documents {
			results[i] = RerankResult{Index: i, Score: 1.0 - float32(i)*0.01}
		}
		if topN > 0 && topN < len(results) {
			return results[:topN], nil
		}
		return results, nil
	}

	// Call SiliconFlow Rerank API
	reqBody := map[string]interface{}{
		"model":     s.model,
		"query":     query,
		"documents": documents,
		"top_n":     topN,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/v1/rerank", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // response body cleanup

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // error body for debugging
		return nil, fmt.Errorf("rerank API error: %s", string(body))
	}

	var result struct {
		Results []struct {
			Index int     `json:"index"`
			Score float32 `json:"relevance_score"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	results := make([]RerankResult, len(result.Results))
	for i, r := range result.Results {
		results[i] = RerankResult{Index: r.Index, Score: r.Score}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}
