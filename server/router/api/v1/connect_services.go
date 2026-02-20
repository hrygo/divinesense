package v1

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
)

// This file contains all Connect service handler method implementations.
// Each method delegates to the underlying gRPC service implementation,
// converting between Connect and gRPC request/response types.

// InstanceService

func (s *ConnectServiceHandler) GetInstanceProfile(ctx context.Context, req *connect.Request[v1pb.GetInstanceProfileRequest]) (*connect.Response[v1pb.InstanceProfile], error) {
	resp, err := s.InstanceService.GetInstanceProfile(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetInstanceSetting(ctx context.Context, req *connect.Request[v1pb.GetInstanceSettingRequest]) (*connect.Response[v1pb.InstanceSetting], error) {
	resp, err := s.InstanceService.GetInstanceSetting(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateInstanceSetting(ctx context.Context, req *connect.Request[v1pb.UpdateInstanceSettingRequest]) (*connect.Response[v1pb.InstanceSetting], error) {
	resp, err := s.InstanceService.UpdateInstanceSetting(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// AuthService
//
// Auth service methods need special handling for response headers (cookies).
// We use connectWithHeaderCarrier helper to inject a header carrier into the context,
// which allows the service to set headers in a protocol-agnostic way.

func (s *ConnectServiceHandler) GetCurrentUser(ctx context.Context, req *connect.Request[v1pb.GetCurrentUserRequest]) (*connect.Response[v1pb.GetCurrentUserResponse], error) {
	return connectWithHeaderCarrier(ctx, func(ctx context.Context) (*v1pb.GetCurrentUserResponse, error) {
		return s.AuthService.GetCurrentUser(ctx, req.Msg)
	})
}

func (s *ConnectServiceHandler) SignIn(ctx context.Context, req *connect.Request[v1pb.SignInRequest]) (*connect.Response[v1pb.SignInResponse], error) {
	return connectWithHeaderCarrier(ctx, func(ctx context.Context) (*v1pb.SignInResponse, error) {
		return s.AuthService.SignIn(ctx, req.Msg)
	})
}

func (s *ConnectServiceHandler) SignOut(ctx context.Context, req *connect.Request[v1pb.SignOutRequest]) (*connect.Response[emptypb.Empty], error) {
	return connectWithHeaderCarrier(ctx, func(ctx context.Context) (*emptypb.Empty, error) {
		return s.AuthService.SignOut(ctx, req.Msg)
	})
}

func (s *ConnectServiceHandler) RefreshToken(ctx context.Context, req *connect.Request[v1pb.RefreshTokenRequest]) (*connect.Response[v1pb.RefreshTokenResponse], error) {
	return connectWithHeaderCarrier(ctx, func(ctx context.Context) (*v1pb.RefreshTokenResponse, error) {
		return s.AuthService.RefreshToken(ctx, req.Msg)
	})
}

// UserService

func (s *ConnectServiceHandler) ListUsers(ctx context.Context, req *connect.Request[v1pb.ListUsersRequest]) (*connect.Response[v1pb.ListUsersResponse], error) {
	resp, err := s.UserService.ListUsers(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetUser(ctx context.Context, req *connect.Request[v1pb.GetUserRequest]) (*connect.Response[v1pb.User], error) {
	resp, err := s.UserService.GetUser(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CreateUser(ctx context.Context, req *connect.Request[v1pb.CreateUserRequest]) (*connect.Response[v1pb.User], error) {
	resp, err := s.UserService.CreateUser(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateUser(ctx context.Context, req *connect.Request[v1pb.UpdateUserRequest]) (*connect.Response[v1pb.User], error) {
	resp, err := s.UserService.UpdateUser(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteUser(ctx context.Context, req *connect.Request[v1pb.DeleteUserRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.UserService.DeleteUser(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListAllUserStats(ctx context.Context, req *connect.Request[v1pb.ListAllUserStatsRequest]) (*connect.Response[v1pb.ListAllUserStatsResponse], error) {
	resp, err := s.UserService.ListAllUserStats(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetUserStats(ctx context.Context, req *connect.Request[v1pb.GetUserStatsRequest]) (*connect.Response[v1pb.UserStats], error) {
	resp, err := s.UserService.GetUserStats(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetUserSetting(ctx context.Context, req *connect.Request[v1pb.GetUserSettingRequest]) (*connect.Response[v1pb.UserSetting], error) {
	resp, err := s.UserService.GetUserSetting(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateUserSetting(ctx context.Context, req *connect.Request[v1pb.UpdateUserSettingRequest]) (*connect.Response[v1pb.UserSetting], error) {
	resp, err := s.UserService.UpdateUserSetting(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListUserSettings(ctx context.Context, req *connect.Request[v1pb.ListUserSettingsRequest]) (*connect.Response[v1pb.ListUserSettingsResponse], error) {
	resp, err := s.UserService.ListUserSettings(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListPersonalAccessTokens(ctx context.Context, req *connect.Request[v1pb.ListPersonalAccessTokensRequest]) (*connect.Response[v1pb.ListPersonalAccessTokensResponse], error) {
	resp, err := s.UserService.ListPersonalAccessTokens(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CreatePersonalAccessToken(ctx context.Context, req *connect.Request[v1pb.CreatePersonalAccessTokenRequest]) (*connect.Response[v1pb.CreatePersonalAccessTokenResponse], error) {
	resp, err := s.UserService.CreatePersonalAccessToken(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeletePersonalAccessToken(ctx context.Context, req *connect.Request[v1pb.DeletePersonalAccessTokenRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.UserService.DeletePersonalAccessToken(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListUserWebhooks(ctx context.Context, req *connect.Request[v1pb.ListUserWebhooksRequest]) (*connect.Response[v1pb.ListUserWebhooksResponse], error) {
	resp, err := s.UserService.ListUserWebhooks(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CreateUserWebhook(ctx context.Context, req *connect.Request[v1pb.CreateUserWebhookRequest]) (*connect.Response[v1pb.UserWebhook], error) {
	resp, err := s.UserService.CreateUserWebhook(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateUserWebhook(ctx context.Context, req *connect.Request[v1pb.UpdateUserWebhookRequest]) (*connect.Response[v1pb.UserWebhook], error) {
	resp, err := s.UserService.UpdateUserWebhook(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteUserWebhook(ctx context.Context, req *connect.Request[v1pb.DeleteUserWebhookRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.UserService.DeleteUserWebhook(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListUserNotifications(ctx context.Context, req *connect.Request[v1pb.ListUserNotificationsRequest]) (*connect.Response[v1pb.ListUserNotificationsResponse], error) {
	resp, err := s.UserService.ListUserNotifications(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateUserNotification(ctx context.Context, req *connect.Request[v1pb.UpdateUserNotificationRequest]) (*connect.Response[v1pb.UserNotification], error) {
	resp, err := s.UserService.UpdateUserNotification(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteUserNotification(ctx context.Context, req *connect.Request[v1pb.DeleteUserNotificationRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.UserService.DeleteUserNotification(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// MemoService

func (s *ConnectServiceHandler) CreateMemo(ctx context.Context, req *connect.Request[v1pb.CreateMemoRequest]) (*connect.Response[v1pb.Memo], error) {
	resp, err := s.MemoService.CreateMemo(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListMemos(ctx context.Context, req *connect.Request[v1pb.ListMemosRequest]) (*connect.Response[v1pb.ListMemosResponse], error) {
	resp, err := s.MemoService.ListMemos(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetMemo(ctx context.Context, req *connect.Request[v1pb.GetMemoRequest]) (*connect.Response[v1pb.Memo], error) {
	resp, err := s.MemoService.GetMemo(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateMemo(ctx context.Context, req *connect.Request[v1pb.UpdateMemoRequest]) (*connect.Response[v1pb.Memo], error) {
	resp, err := s.MemoService.UpdateMemo(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteMemo(ctx context.Context, req *connect.Request[v1pb.DeleteMemoRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.MemoService.DeleteMemo(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) SetMemoAttachments(ctx context.Context, req *connect.Request[v1pb.SetMemoAttachmentsRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.MemoService.SetMemoAttachments(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListMemoAttachments(ctx context.Context, req *connect.Request[v1pb.ListMemoAttachmentsRequest]) (*connect.Response[v1pb.ListMemoAttachmentsResponse], error) {
	resp, err := s.MemoService.ListMemoAttachments(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) SetMemoRelations(ctx context.Context, req *connect.Request[v1pb.SetMemoRelationsRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.MemoService.SetMemoRelations(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListMemoRelations(ctx context.Context, req *connect.Request[v1pb.ListMemoRelationsRequest]) (*connect.Response[v1pb.ListMemoRelationsResponse], error) {
	resp, err := s.MemoService.ListMemoRelations(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CreateMemoComment(ctx context.Context, req *connect.Request[v1pb.CreateMemoCommentRequest]) (*connect.Response[v1pb.Memo], error) {
	resp, err := s.MemoService.CreateMemoComment(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListMemoComments(ctx context.Context, req *connect.Request[v1pb.ListMemoCommentsRequest]) (*connect.Response[v1pb.ListMemoCommentsResponse], error) {
	resp, err := s.MemoService.ListMemoComments(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListMemoReactions(ctx context.Context, req *connect.Request[v1pb.ListMemoReactionsRequest]) (*connect.Response[v1pb.ListMemoReactionsResponse], error) {
	resp, err := s.MemoService.ListMemoReactions(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpsertMemoReaction(ctx context.Context, req *connect.Request[v1pb.UpsertMemoReactionRequest]) (*connect.Response[v1pb.Reaction], error) {
	resp, err := s.MemoService.UpsertMemoReaction(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteMemoReaction(ctx context.Context, req *connect.Request[v1pb.DeleteMemoReactionRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.MemoService.DeleteMemoReaction(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// AttachmentService

func (s *ConnectServiceHandler) CreateAttachment(ctx context.Context, req *connect.Request[v1pb.CreateAttachmentRequest]) (*connect.Response[v1pb.Attachment], error) {
	resp, err := s.AttachmentService.CreateAttachment(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListAttachments(ctx context.Context, req *connect.Request[v1pb.ListAttachmentsRequest]) (*connect.Response[v1pb.ListAttachmentsResponse], error) {
	resp, err := s.AttachmentService.ListAttachments(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetAttachment(ctx context.Context, req *connect.Request[v1pb.GetAttachmentRequest]) (*connect.Response[v1pb.Attachment], error) {
	resp, err := s.AttachmentService.GetAttachment(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateAttachment(ctx context.Context, req *connect.Request[v1pb.UpdateAttachmentRequest]) (*connect.Response[v1pb.Attachment], error) {
	resp, err := s.AttachmentService.UpdateAttachment(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteAttachment(ctx context.Context, req *connect.Request[v1pb.DeleteAttachmentRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.AttachmentService.DeleteAttachment(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// ShortcutService

func (s *ConnectServiceHandler) ListShortcuts(ctx context.Context, req *connect.Request[v1pb.ListShortcutsRequest]) (*connect.Response[v1pb.ListShortcutsResponse], error) {
	resp, err := s.ShortcutService.ListShortcuts(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetShortcut(ctx context.Context, req *connect.Request[v1pb.GetShortcutRequest]) (*connect.Response[v1pb.Shortcut], error) {
	resp, err := s.ShortcutService.GetShortcut(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CreateShortcut(ctx context.Context, req *connect.Request[v1pb.CreateShortcutRequest]) (*connect.Response[v1pb.Shortcut], error) {
	resp, err := s.ShortcutService.CreateShortcut(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateShortcut(ctx context.Context, req *connect.Request[v1pb.UpdateShortcutRequest]) (*connect.Response[v1pb.Shortcut], error) {
	resp, err := s.ShortcutService.UpdateShortcut(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteShortcut(ctx context.Context, req *connect.Request[v1pb.DeleteShortcutRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.ShortcutService.DeleteShortcut(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// ActivityService

func (s *ConnectServiceHandler) ListActivities(ctx context.Context, req *connect.Request[v1pb.ListActivitiesRequest]) (*connect.Response[v1pb.ListActivitiesResponse], error) {
	resp, err := s.ActivityService.ListActivities(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetActivity(ctx context.Context, req *connect.Request[v1pb.GetActivityRequest]) (*connect.Response[v1pb.Activity], error) {
	resp, err := s.ActivityService.GetActivity(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// IdentityProviderService

func (s *ConnectServiceHandler) ListIdentityProviders(ctx context.Context, req *connect.Request[v1pb.ListIdentityProvidersRequest]) (*connect.Response[v1pb.ListIdentityProvidersResponse], error) {
	resp, err := s.IdentityProviderService.ListIdentityProviders(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetIdentityProvider(ctx context.Context, req *connect.Request[v1pb.GetIdentityProviderRequest]) (*connect.Response[v1pb.IdentityProvider], error) {
	resp, err := s.IdentityProviderService.GetIdentityProvider(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) CreateIdentityProvider(ctx context.Context, req *connect.Request[v1pb.CreateIdentityProviderRequest]) (*connect.Response[v1pb.IdentityProvider], error) {
	resp, err := s.IdentityProviderService.CreateIdentityProvider(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) UpdateIdentityProvider(ctx context.Context, req *connect.Request[v1pb.UpdateIdentityProviderRequest]) (*connect.Response[v1pb.IdentityProvider], error) {
	resp, err := s.IdentityProviderService.UpdateIdentityProvider(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) DeleteIdentityProvider(ctx context.Context, req *connect.Request[v1pb.DeleteIdentityProviderRequest]) (*connect.Response[emptypb.Empty], error) {
	resp, err := s.IdentityProviderService.DeleteIdentityProvider(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

// AIService - Session Stats methods

func (s *ConnectServiceHandler) GetSessionStats(ctx context.Context, req *connect.Request[v1pb.GetSessionStatsRequest]) (*connect.Response[v1pb.SessionStats], error) {
	resp, err := s.AIService.GetSessionStats(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) ListSessionStats(ctx context.Context, req *connect.Request[v1pb.ListSessionStatsRequest]) (*connect.Response[v1pb.ListSessionStatsResponse], error) {
	resp, err := s.AIService.ListSessionStats(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetCostStats(ctx context.Context, req *connect.Request[v1pb.GetCostStatsRequest]) (*connect.Response[v1pb.CostStats], error) {
	resp, err := s.AIService.GetCostStats(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) GetUserCostSettings(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1pb.UserCostSettings], error) {
	resp, err := s.AIService.GetUserCostSettings(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}

func (s *ConnectServiceHandler) SetUserCostSettings(ctx context.Context, req *connect.Request[v1pb.SetUserCostSettingsRequest]) (*connect.Response[v1pb.UserCostSettings], error) {
	resp, err := s.AIService.SetUserCostSettings(ctx, req.Msg)
	if err != nil {
		return nil, convertGRPCError(err)
	}
	return connect.NewResponse(resp), nil
}
