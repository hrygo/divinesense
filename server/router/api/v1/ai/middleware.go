package ai

import (
	"context"
	"fmt"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hrygo/divinesense/plugin/ai"
	"github.com/hrygo/divinesense/server/auth"
	"github.com/hrygo/divinesense/server/internal/errors"
	"github.com/hrygo/divinesense/server/middleware"
)

// ChatRequest represents a chat request.
type ChatRequest struct {
	Message            string
	AgentType          AgentType
	Timezone           string
	DeviceContext      string
	History            []string
	UserID             int32
	ConversationID     int32
	IsTempConversation bool
	GeekMode           bool
	EvolutionMode      bool
}

// Handler is the interface for handling chat requests.
type Handler interface {
	Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error
}

// Middleware is a function that wraps a handler.
type Middleware func(Handler) Handler

// Chain chains multiple middlewares together.
func Chain(h Handler, middlewares ...Middleware) Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// ValidationMiddleware validates incoming requests.
type ValidationMiddleware struct{}

// NewValidationMiddleware creates a new validation middleware.
func NewValidationMiddleware() Middleware {
	return func(next Handler) Handler {
		return &validationHandler{next: next}
	}
}

type validationHandler struct {
	next Handler
}

func (h *validationHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	if req.Message == "" {
		return status.Error(codes.InvalidArgument, "message is required")
	}
	return h.next.Handle(ctx, req, stream)
}

// AuthMiddleware authenticates the user.
type AuthMiddleware struct{}

// NewAuthMiddleware creates a new auth middleware.
func NewAuthMiddleware() Middleware {
	return func(next Handler) Handler {
		return &authHandler{next: next}
	}
}

type authHandler struct {
	next Handler
}

func (h *authHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	// Get user ID from context (set by authentication layer)
	userID, err := getUserID(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, "unauthorized")
	}

	req.UserID = userID
	return h.next.Handle(ctx, req, stream)
}

// RateLimitMiddleware applies rate limiting.
// Note: The struct is not preserved as it is only used to create the handler closure.

// NewRateLimitMiddleware creates a new rate limit middleware.
func NewRateLimitMiddleware(limiter *middleware.RateLimiter) Middleware {
	return func(next Handler) Handler {
		return &rateLimitHandler{
			limiter: limiter,
			next:    next,
		}
	}
}

type rateLimitHandler struct {
	limiter *middleware.RateLimiter
	next    Handler
}

func (h *rateLimitHandler) Handle(ctx context.Context, req *ChatRequest, stream ChatStream) error {
	userKey := strconv.FormatInt(int64(req.UserID), 10)
	if !h.limiter.Allow(userKey) {
		return status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}
	return h.next.Handle(ctx, req, stream)
}

// TruncateString truncates a string for logging.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// getUserID extracts the user ID from context using the auth layer.
func getUserID(ctx context.Context) (int32, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return 0, fmt.Errorf("user ID not found in context")
	}
	return userID, nil
}

// ToAIError converts a gRPC status error to an AIError.
func ToAIError(err error) *errors.AIError {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return errors.Wrap(err, errors.ErrCodeServiceUnavailable, "unknown error")
	}

	switch st.Code() {
	case codes.Unauthenticated:
		return errors.Unauthorized(st.Message())
	case codes.ResourceExhausted:
		return errors.RateLimitExceeded(st.Message())
	case codes.InvalidArgument:
		return errors.InvalidArgument(st.Message())
	case codes.Unavailable:
		return errors.ServiceUnavailable(st.Message())
	case codes.DeadlineExceeded:
		return errors.Timeout(st.Message())
	case codes.Canceled:
		return errors.ContextCanceled(err)
	default:
		return errors.Wrap(err, errors.ErrCodeAgentExecutionFailed, st.Message())
	}
}

// FromAIError converts an AIError to a gRPC status error.
func FromAIError(err *errors.AIError) error {
	if err == nil {
		return nil
	}

	switch err.Code {
	case errors.ErrCodeUnauthorized:
		return status.Error(codes.Unauthenticated, err.Message)
	case errors.ErrCodeRateLimitExceeded:
		return status.Error(codes.ResourceExhausted, err.Message)
	case errors.ErrCodeInvalidArgument:
		return status.Error(codes.InvalidArgument, err.Message)
	case errors.ErrCodeServiceUnavailable, errors.ErrCodeLLMUnavailable:
		return status.Error(codes.Unavailable, err.Message)
	case errors.ErrCodeTimeout:
		return status.Error(codes.DeadlineExceeded, err.Message)
	case errors.ErrCodeContextCanceled:
		return status.Error(codes.Canceled, err.Message)
	default:
		return status.Error(codes.Internal, err.Message)
	}
}

// BuildChatMessages constructs chat messages from user input and history.
func BuildChatMessages(message string, history []string, systemPrompt string) []ai.Message {
	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
	}

	// Add history (skip empty messages to avoid LLM API errors)
	for i := 0; i < len(history)-1; i += 2 {
		if i+1 < len(history) {
			userMsg := history[i]
			assistantMsg := history[i+1]
			// Only add non-empty messages
			if userMsg != "" && assistantMsg != "" {
				messages = append(messages, ai.Message{Role: "user", Content: userMsg})
				messages = append(messages, ai.Message{Role: "assistant", Content: assistantMsg})
			}
		}
	}

	// Add current user message
	messages = append(messages, ai.Message{Role: "user", Content: message})

	return messages
}
