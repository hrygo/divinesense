package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Result represents a reranking result.
type Result struct {
	Index int     // Original index
	Score float32 // Relevance score
}

// Service is the reranking service interface.
type Service interface {
	// Rerank reorders documents by relevance.
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]Result, error)

	// IsEnabled returns whether the service is enabled.
	IsEnabled() bool
}

// Config represents reranker service configuration.
type Config struct {
	Provider string
	Model    string
	APIKey   string
	BaseURL  string
	Enabled  bool
}

type service struct {
	client  *http.Client
	apiKey  string
	baseURL string
	model   string
	enabled bool
}

// NewService creates a new reranker Service.
func NewService(cfg *Config) Service {
	return &service{
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

func (s *service) IsEnabled() bool {
	return s.enabled
}

func (s *service) Rerank(ctx context.Context, query string, documents []string, topN int) ([]Result, error) {
	if !s.enabled {
		// Return original order when disabled
		results := make([]Result, len(documents))
		for i := range documents {
			results[i] = Result{Index: i, Score: 1.0 - float32(i)*0.01}
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

	baseURL := strings.TrimRight(s.baseURL, "/")
	if strings.HasSuffix(baseURL, "/v1") {
		baseURL += "/rerank"
	} else {
		baseURL += "/v1/rerank"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("rerank API error: HTTP %d", resp.StatusCode)
		}
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

	results := make([]Result, len(result.Results))
	for i, r := range result.Results {
		results[i] = Result{Index: r.Index, Score: r.Score}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}
