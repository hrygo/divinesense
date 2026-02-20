package v1

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

func (s *UserService) ListUserWebhooks(ctx context.Context, request *v1pb.ListUserWebhooksRequest) (*v1pb.ListUserWebhooksResponse, error) {
	userID, err := ExtractUserIDFromName(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid parent: %v", err)
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if currentUser.ID != userID && currentUser.Role != store.RoleHost && currentUser.Role != store.RoleAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	webhooks, err := s.Store.GetUserWebhooks(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user webhooks: %v", err)
	}

	userWebhooks := make([]*v1pb.UserWebhook, 0, len(webhooks))
	for _, webhook := range webhooks {
		userWebhooks = append(userWebhooks, convertUserWebhookFromUserSetting(webhook, userID))
	}

	return &v1pb.ListUserWebhooksResponse{
		Webhooks: userWebhooks,
	}, nil
}

func (s *UserService) CreateUserWebhook(ctx context.Context, request *v1pb.CreateUserWebhookRequest) (*v1pb.UserWebhook, error) {
	userID, err := ExtractUserIDFromName(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid parent: %v", err)
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if currentUser.ID != userID && currentUser.Role != store.RoleHost && currentUser.Role != store.RoleAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	if request.Webhook.Url == "" {
		return nil, status.Errorf(codes.InvalidArgument, "webhook URL is required")
	}

	webhookID := generateUserWebhookID()
	webhook := &storepb.WebhooksUserSetting_Webhook{
		Id:    webhookID,
		Title: request.Webhook.DisplayName,
		Url:   strings.TrimSpace(request.Webhook.Url),
	}

	err = s.Store.AddUserWebhook(ctx, userID, webhook)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create webhook: %v", err)
	}

	return convertUserWebhookFromUserSetting(webhook, userID), nil
}

func (s *UserService) UpdateUserWebhook(ctx context.Context, request *v1pb.UpdateUserWebhookRequest) (*v1pb.UserWebhook, error) {
	if request.Webhook == nil {
		return nil, status.Errorf(codes.InvalidArgument, "webhook is required")
	}

	webhookID, userID, err := parseUserWebhookName(request.Webhook.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid webhook name: %v", err)
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if currentUser.ID != userID && currentUser.Role != store.RoleHost && currentUser.Role != store.RoleAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	// Get existing webhooks
	webhooks, err := s.Store.GetUserWebhooks(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user webhooks: %v", err)
	}

	// Find the webhook to update
	var targetWebhook *storepb.WebhooksUserSetting_Webhook
	for _, webhook := range webhooks {
		if webhook.Id == webhookID {
			targetWebhook = webhook
			break
		}
	}

	if targetWebhook == nil {
		return nil, status.Errorf(codes.NotFound, "webhook not found")
	}

	// Update the webhook
	updatedWebhook := &storepb.WebhooksUserSetting_Webhook{
		Id:    webhookID,
		Title: targetWebhook.Title,
		Url:   targetWebhook.Url,
	}

	if request.UpdateMask != nil {
		for _, path := range request.UpdateMask.Paths {
			switch path {
			case "url":
				if request.Webhook.Url != "" {
					updatedWebhook.Url = strings.TrimSpace(request.Webhook.Url)
				}
			case "display_name":
				updatedWebhook.Title = request.Webhook.DisplayName
			default:
				// Ignore unsupported fields
			}
		}
	} else {
		// If no update mask is provided, update all fields
		if request.Webhook.Url != "" {
			updatedWebhook.Url = strings.TrimSpace(request.Webhook.Url)
		}
		updatedWebhook.Title = request.Webhook.DisplayName
	}

	err = s.Store.UpdateUserWebhook(ctx, userID, updatedWebhook)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update webhook: %v", err)
	}

	return convertUserWebhookFromUserSetting(updatedWebhook, userID), nil
}

func (s *UserService) DeleteUserWebhook(ctx context.Context, request *v1pb.DeleteUserWebhookRequest) (*emptypb.Empty, error) {
	webhookID, userID, err := parseUserWebhookName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid webhook name: %v", err)
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if currentUser.ID != userID && currentUser.Role != store.RoleHost && currentUser.Role != store.RoleAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	// Get existing webhooks to verify the webhook exists
	webhooks, err := s.Store.GetUserWebhooks(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user webhooks: %v", err)
	}

	// Check if webhook exists
	found := false
	for _, webhook := range webhooks {
		if webhook.Id == webhookID {
			found = true
			break
		}
	}

	if !found {
		return nil, status.Errorf(codes.NotFound, "webhook not found")
	}

	err = s.Store.RemoveUserWebhook(ctx, userID, webhookID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete webhook: %v", err)
	}

	return &emptypb.Empty{}, nil
}
