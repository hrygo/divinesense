package v1

import (
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

func getDefaultUserGeneralSetting() *v1pb.UserSetting_GeneralSetting {
	return &v1pb.UserSetting_GeneralSetting{
		Locale:         "en",
		MemoVisibility: "PRIVATE",
		Theme:          "",
	}
}

func (s *UserService) GetUserSetting(ctx context.Context, request *v1pb.GetUserSettingRequest) (*v1pb.UserSetting, error) {
	// Parse resource name: users/{user}/settings/{setting}
	userID, settingKey, err := ExtractUserIDAndSettingKeyFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid resource name: %v", err)
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	// Only allow user to get their own settings
	if currentUser.ID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	// Convert setting key string to store enum
	storeKey, err := convertSettingKeyToStore(settingKey)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid setting key: %v", err)
	}

	userSetting, err := s.Store.GetUserSetting(ctx, &store.FindUserSetting{
		UserID: &userID,
		Key:    storeKey,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user setting: %v", err)
	}

	return convertUserSettingFromStore(userSetting, userID, storeKey), nil
}

func (s *UserService) UpdateUserSetting(ctx context.Context, request *v1pb.UpdateUserSettingRequest) (*v1pb.UserSetting, error) {
	// Parse resource name: users/{user}/settings/{setting}
	userID, settingKey, err := ExtractUserIDAndSettingKeyFromName(request.Setting.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid resource name: %v", err)
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	// Only allow user to update their own settings
	if currentUser.ID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update mask is empty")
	}

	// Convert setting key string to store enum
	storeKey, err := convertSettingKeyToStore(settingKey)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid setting key: %v", err)
	}

	// Only GENERAL settings are supported via UpdateUserSetting
	// Other setting types have dedicated service methods
	if storeKey != storepb.UserSetting_GENERAL {
		return nil, status.Errorf(codes.InvalidArgument, "setting type %s should not be updated via UpdateUserSetting", storeKey.String())
	}

	existingUserSetting, _ := s.Store.GetUserSetting(ctx, &store.FindUserSetting{
		UserID: &userID,
		Key:    storeKey,
	})

	generalSetting := &storepb.GeneralUserSetting{}
	if existingUserSetting != nil {
		// Start with existing general setting values
		generalSetting = existingUserSetting.GetGeneral()
	}

	updatedGeneral := &v1pb.UserSetting_GeneralSetting{
		MemoVisibility: generalSetting.GetMemoVisibility(),
		Locale:         generalSetting.GetLocale(),
		Theme:          generalSetting.GetTheme(),
	}

	// Apply updates for fields specified in the update mask
	incomingGeneral := request.Setting.GetGeneralSetting()
	for _, field := range request.UpdateMask.Paths {
		switch field {
		case "memo_visibility":
			updatedGeneral.MemoVisibility = incomingGeneral.MemoVisibility
		case "theme":
			updatedGeneral.Theme = incomingGeneral.Theme
		case "locale":
			updatedGeneral.Locale = incomingGeneral.Locale
		default:
			// Ignore unsupported fields
		}
	}

	// Create the updated setting
	updatedSetting := &v1pb.UserSetting{
		Name: request.Setting.Name,
		Value: &v1pb.UserSetting_GeneralSetting_{
			GeneralSetting: updatedGeneral,
		},
	}

	// Convert API setting to store setting
	storeSetting, err := convertUserSettingToStore(updatedSetting, userID, storeKey)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to convert setting: %v", err)
	}

	// Upsert the setting
	if _, err := s.Store.UpsertUserSetting(ctx, storeSetting); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upsert user setting: %v", err)
	}

	return s.GetUserSetting(ctx, &v1pb.GetUserSettingRequest{Name: request.Setting.Name})
}

func (s *UserService) ListUserSettings(ctx context.Context, request *v1pb.ListUserSettingsRequest) (*v1pb.ListUserSettingsResponse, error) {
	userID, err := ExtractUserIDFromName(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid parent name: %v", err)
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get current user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	// Only allow user to list their own settings
	if currentUser.ID != userID {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	userSettings, err := s.Store.ListUserSettings(ctx, &store.FindUserSetting{
		UserID: &userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list user settings: %v", err)
	}

	settings := make([]*v1pb.UserSetting, 0, len(userSettings))
	for _, storeSetting := range userSettings {
		apiSetting := convertUserSettingFromStore(storeSetting, userID, storeSetting.Key)
		if apiSetting != nil {
			settings = append(settings, apiSetting)
		}
	}

	// If no general setting exists, add a default one
	hasGeneral := false
	for _, setting := range settings {
		if setting.GetGeneralSetting() != nil {
			hasGeneral = true
			break
		}
	}
	if !hasGeneral {
		defaultGeneral := &v1pb.UserSetting{
			Name: fmt.Sprintf("users/%d/settings/general", userID),
			Value: &v1pb.UserSetting_GeneralSetting_{
				GeneralSetting: getDefaultUserGeneralSetting(),
			},
		}
		settings = append([]*v1pb.UserSetting{defaultGeneral}, settings...)
	}

	response := &v1pb.ListUserSettingsResponse{
		Settings:  settings,
		TotalSize: int32(len(settings)),
	}

	return response, nil
}
