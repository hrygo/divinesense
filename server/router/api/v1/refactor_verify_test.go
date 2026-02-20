package v1

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/stretchr/testify/assert"
)

// Minimal mock to test specific methods without pulling in the entire world
type MockConnectHandler struct {
	AIService AIServiceInterface
}

func (s *MockConnectHandler) requireAI() error {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	return nil
}

func (s *MockConnectHandler) SuggestTags(ctx context.Context, req *connect.Request[v1pb.SuggestTagsRequest]) (*connect.Response[v1pb.SuggestTagsResponse], error) {
	// --- Logic under test START ---
	if err := s.requireAI(); err != nil {
		return nil, err
	}
	// --- Logic under test END ---

	return connect.NewResponse(&v1pb.SuggestTagsResponse{}), nil
}

// Define local interface to match what's used
type AIServiceInterface interface {
	IsEnabled() bool
	// SuggestTags(ctx context.Context, req *v1pb.SuggestTagsRequest) (*v1pb.SuggestTagsResponse, error)
	// Add other methods if needed
}

func TestSuggestTags_AI_Check_Behavior(t *testing.T) {
	t.Run("AIService is nil", func(t *testing.T) {
		handler := &MockConnectHandler{
			AIService: nil,
		}

		req := connect.NewRequest(&v1pb.SuggestTagsRequest{})
		_, err := handler.SuggestTags(context.Background(), req)

		assert.Error(t, err)
		assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
		assert.Contains(t, err.Error(), "AI features are disabled")
	})
}
