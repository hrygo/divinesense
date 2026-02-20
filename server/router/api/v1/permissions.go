package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hrygo/divinesense/store"
)

// requireUserAccess ensures that the context contains an authenticated user,
// and that user is either the target user or a superuser (Admin/Host).
func (s *APIV1Service) requireUserAccess(ctx context.Context, targetUserID int32) (*store.User, error) {
	currentUser, err := s.fetchCurrentUser(ctx)
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

// requireAI ensures that the AI service is enabled and available.
func (s *ConnectServiceHandler) requireAI() error {
	if err := s.requireAI(); err != nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("AI features are disabled"))
	}
	return nil
}
