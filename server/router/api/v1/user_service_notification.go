package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

// ListUserNotifications lists all notifications for a user.
// Notifications are backed by the inbox storage layer and represent activities
// that require user attention (e.g., memo comments).
func (s *UserService) ListUserNotifications(ctx context.Context, request *v1pb.ListUserNotificationsRequest) (*v1pb.ListUserNotificationsResponse, error) {
	userID, err := ExtractUserIDFromName(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user name: %v", err)
	}

	// Verify the requesting user has permission to view these notifications
	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if currentUser.ID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	// Fetch inbox items from storage
	// Filter at database level to only include MEMO_COMMENT notifications (ignore legacy VERSION_UPDATE entries)
	memoCommentType := storepb.InboxMessage_MEMO_COMMENT
	inboxes, err := s.Store.ListInboxes(ctx, &store.FindInbox{
		ReceiverID:  &userID,
		MessageType: &memoCommentType,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list inboxes: %v", err)
	}

	// Convert storage layer inboxes to API notifications
	notifications := []*v1pb.UserNotification{}
	for _, inbox := range inboxes {
		notification, err := s.convertInboxToUserNotification(ctx, inbox)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert inbox: %v", err)
		}
		notifications = append(notifications, notification)
	}

	return &v1pb.ListUserNotificationsResponse{
		Notifications: notifications,
	}, nil
}

// UpdateUserNotification updates a notification's status (e.g., marking as read/archived).
// Only the notification owner can update their notifications.
func (s *UserService) UpdateUserNotification(ctx context.Context, request *v1pb.UpdateUserNotificationRequest) (*v1pb.UserNotification, error) {
	if request.Notification == nil {
		return nil, status.Errorf(codes.InvalidArgument, "notification is required")
	}

	notificationID, err := ExtractNotificationIDFromName(request.Notification.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid notification name: %v", err)
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}

	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	// Verify ownership before updating
	inboxes, err := s.Store.ListInboxes(ctx, &store.FindInbox{
		ID: &notificationID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get inbox: %v", err)
	}
	if len(inboxes) == 0 {
		return nil, status.Errorf(codes.NotFound, "notification not found")
	}
	inbox := inboxes[0]
	if inbox.ReceiverID != currentUser.ID {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	// Build update request based on field mask
	update := &store.UpdateInbox{
		ID: notificationID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "status":
			// Convert API status enum to storage enum
			var inboxStatus store.InboxStatus
			switch request.Notification.Status {
			case v1pb.UserNotification_UNREAD:
				inboxStatus = store.UNREAD
			case v1pb.UserNotification_ARCHIVED:
				inboxStatus = store.ARCHIVED
			default:
				return nil, status.Errorf(codes.InvalidArgument, "invalid status")
			}
			update.Status = inboxStatus
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid update path: %s", path)
		}
	}

	updatedInbox, err := s.Store.UpdateInbox(ctx, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update inbox: %v", err)
	}

	notification, err := s.convertInboxToUserNotification(ctx, updatedInbox)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert inbox: %v", err)
	}

	return notification, nil
}

// DeleteUserNotification permanently deletes a notification.
// Only the notification owner can delete their notifications.
func (s *UserService) DeleteUserNotification(ctx context.Context, request *v1pb.DeleteUserNotificationRequest) (*emptypb.Empty, error) {
	notificationID, err := ExtractNotificationIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid notification name: %v", err)
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}

	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	// Verify ownership before deletion
	inboxes, err := s.Store.ListInboxes(ctx, &store.FindInbox{
		ID: &notificationID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get inbox: %v", err)
	}
	if len(inboxes) == 0 {
		return nil, status.Errorf(codes.NotFound, "notification not found")
	}
	inbox := inboxes[0]
	if inbox.ReceiverID != currentUser.ID {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	if err := s.Store.DeleteInbox(ctx, &store.DeleteInbox{
		ID: notificationID,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete inbox: %v", err)
	}

	return &emptypb.Empty{}, nil
}
