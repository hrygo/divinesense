package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hrygo/divinesense/server/auth"
	"github.com/hrygo/divinesense/store"
)

// fetchCurrentUser retrieves the authenticated user from the context.
// Returns (nil, nil) if no user ID is found in context (anonymous request).
// This is a standalone function to decouple from the god struct.
func fetchCurrentUser(ctx context.Context, s *store.Store) (*store.User, error) {
	userID := auth.GetUserID(ctx)
	if userID == 0 {
		return nil, nil
	}
	user, err := s.GetUser(ctx, &store.FindUser{
		ID: &userID,
	})
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.Errorf("user %d not found", userID)
	}
	return user, nil
}

// requireUserAccess ensures that the context contains an authenticated user,
// and that user is either the target user or a superuser (Admin/Host).
// This is a standalone function to decouple from the god struct.
func requireUserAccess(ctx context.Context, s *store.Store, targetUserID int32) (*store.User, error) {
	currentUser, err := fetchCurrentUser(ctx, s)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	// Verify permission: Allow if user is target, or if user is superuser
	if currentUser.ID != targetUserID && currentUser.Role != store.RoleAdmin && currentUser.Role != store.RoleHost {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	return currentUser, nil
}

func (s *ConnectServiceHandler) requireAI() error {
	if s.AIService == nil || !s.AIService.IsEnabled() {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	return nil
}
