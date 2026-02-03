package ai

import (
	"context"
	"errors"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// EmbeddingService is the vector embedding service interface.
type EmbeddingService interface {
	// Embed generates vector for a single text.
	Embed(ctx context.Context, text string) ([]float32, error)

	// EmbedBatch generates vectors for multiple texts.
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)

	// Dimensions returns the vector dimension.
	Dimensions() int
}

type embeddingService struct {
	client     *openai.Client
	model      string
	dimensions int
}

// NewEmbeddingService creates a new EmbeddingService.
func NewEmbeddingService(cfg *EmbeddingConfig) (EmbeddingService, error) {
	var clientConfig openai.ClientConfig

	switch cfg.Provider {
	case "siliconflow":
		// SiliconFlow is compatible with OpenAI API
		clientConfig = openai.DefaultConfig(cfg.APIKey)
		if cfg.BaseURL != "" {
			clientConfig.BaseURL = cfg.BaseURL
		}

	case "openai":
		clientConfig = openai.DefaultConfig(cfg.APIKey)
		if cfg.BaseURL != "" {
			clientConfig.BaseURL = cfg.BaseURL
		}

	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}

	client := openai.NewClientWithConfig(clientConfig)

	return &embeddingService{
		client:     client,
		model:      cfg.Model,
		dimensions: cfg.Dimensions,
	}, nil
}

func (s *embeddingService) Embed(ctx context.Context, text string) ([]float32, error) {
	vectors, err := s.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, errors.New("empty embedding result")
	}
	return vectors[0], nil
}

func (s *embeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, errors.New("no texts provided for embedding")
	}

	req := openai.EmbeddingRequest{
		Input:      texts,
		Model:      openai.EmbeddingModel(s.model),
		Dimensions: s.dimensions,
	}

	resp, err := s.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create embeddings failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("empty embedding response")
	}

	// Extract vectors from response
	vectors := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		vectors[i] = data.Embedding
	}

	return vectors, nil
}

func (s *embeddingService) Dimensions() int {
	return s.dimensions
}
