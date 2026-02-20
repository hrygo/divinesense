package v1

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/hrygo/divinesense/internal/base"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/store"
)

type UserService struct {
	v1pb.UnimplementedUserServiceServer
	Store *store.Store
}

func (s *UserService) ListUsers(ctx context.Context, request *v1pb.ListUsersRequest) (*v1pb.ListUsersResponse, error) {
	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	if currentUser.Role != store.RoleHost && currentUser.Role != store.RoleAdmin {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	userFind := &store.FindUser{}

	if request.Filter != "" {
		username, err := extractUsernameFromFilter(request.Filter)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
		}
		if username != "" {
			userFind.Username = &username
		}
	}

	users, err := s.Store.ListUsers(ctx, userFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	// TODO: Implement proper ordering, and pagination
	// For now, return all users with basic structure
	response := &v1pb.ListUsersResponse{
		Users:     []*v1pb.User{},
		TotalSize: int32(len(users)),
	}
	for _, user := range users {
		response.Users = append(response.Users, convertUserFromStore(user))
	}
	return response, nil
}

func (s *UserService) GetUser(ctx context.Context, request *v1pb.GetUserRequest) (*v1pb.User, error) {
	// Extract identifier from "users/{id_or_username}"
	identifier := extractUserIdentifierFromName(request.Name)
	if identifier == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user name: %s", request.Name)
	}

	var user *store.User
	var err error

	// Try to parse as numeric ID first
	if userID, parseErr := strconv.ParseInt(identifier, 10, 32); parseErr == nil {
		// It's a numeric ID
		userID32 := int32(userID)
		user, err = s.Store.GetUser(ctx, &store.FindUser{
			ID: &userID32,
		})
	} else {
		// It's a username
		user, err = s.Store.GetUser(ctx, &store.FindUser{
			Username: &identifier,
		})
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}
	return convertUserFromStore(user), nil
}

func (s *UserService) CreateUser(ctx context.Context, request *v1pb.CreateUserRequest) (*v1pb.User, error) {
	// Get current user (might be nil for unauthenticated requests)
	currentUser, _ := fetchCurrentUser(ctx, s.Store)

	// Check if there are any existing users (for first-time setup detection)
	limitOne := 1
	allUsers, err := s.Store.ListUsers(ctx, &store.FindUser{Limit: &limitOne})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}
	isFirstUser := len(allUsers) == 0

	// Check registration settings FIRST (unless it's the very first user)
	if !isFirstUser {
		// Only allow user registration if it is enabled in the settings, or if the user is a superuser
		if currentUser == nil || !isSuperUser(currentUser) {
			instanceGeneralSetting, err := s.Store.GetInstanceGeneralSetting(ctx)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get instance general setting, error: %v", err)
			}
			if instanceGeneralSetting.DisallowUserRegistration {
				return nil, status.Errorf(codes.PermissionDenied, "user registration is not allowed")
			}
		}
	}

	// Determine the role to assign
	var roleToAssign store.Role
	if isFirstUser {
		// First-time setup: create the first user as HOST (no authentication required)
		roleToAssign = store.RoleHost
	} else if currentUser != nil && currentUser.Role == store.RoleHost {
		// Authenticated HOST user can create users with any role specified in request
		if request.User.Role != v1pb.User_ROLE_UNSPECIFIED {
			roleToAssign = convertUserRoleToStore(request.User.Role)
		} else {
			roleToAssign = store.RoleUser
		}
	} else {
		// Unauthenticated or non-HOST users can only create normal users
		roleToAssign = store.RoleUser
	}

	if !base.UIDMatcher.MatchString(strings.ToLower(request.User.Username)) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid username: %s", request.User.Username)
	}

	// If validate_only is true, just validate without creating
	if request.ValidateOnly {
		// Perform validation checks without actually creating the user
		return &v1pb.User{
			Username:    request.User.Username,
			Email:       request.User.Email,
			DisplayName: request.User.DisplayName,
			Role:        convertUserRoleFromStore(roleToAssign),
		}, nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.User.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "failed to generate password hash").SetInternal(err)
	}

	user, err := s.Store.CreateUser(ctx, &store.User{
		Username:     request.User.Username,
		Role:         roleToAssign,
		Email:        request.User.Email,
		Nickname:     request.User.DisplayName,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return convertUserFromStore(user), nil
}

func (s *UserService) UpdateUser(ctx context.Context, request *v1pb.UpdateUserRequest) (*v1pb.User, error) {
	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update mask is empty")
	}
	userID, err := ExtractUserIDFromName(request.User.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user name: %v", err)
	}
	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if currentUser == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	// Check permission.
	// Only allow admin or self to update user.
	if currentUser.ID != userID && currentUser.Role != store.RoleAdmin && currentUser.Role != store.RoleHost {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	user, err := s.Store.GetUser(ctx, &store.FindUser{ID: &userID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		// Handle allow_missing field
		if request.AllowMissing {
			// Could create user if missing, but for now return not found
			return nil, status.Errorf(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	currentTs := time.Now().Unix()
	update := &store.UpdateUser{
		ID:        user.ID,
		UpdatedTs: &currentTs,
	}
	instanceGeneralSetting, err := s.Store.GetInstanceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get instance general setting: %v", err)
	}
	for _, field := range request.UpdateMask.Paths {
		switch field {
		case "username":
			if instanceGeneralSetting.DisallowChangeUsername {
				return nil, status.Errorf(codes.PermissionDenied, "permission denied: disallow change username")
			}
			if !base.UIDMatcher.MatchString(strings.ToLower(request.User.Username)) {
				return nil, status.Errorf(codes.InvalidArgument, "invalid username: %s", request.User.Username)
			}
			update.Username = &request.User.Username
		case "display_name":
			if instanceGeneralSetting.DisallowChangeNickname {
				return nil, status.Errorf(codes.PermissionDenied, "permission denied: disallow change nickname")
			}
			update.Nickname = &request.User.DisplayName
		case "email":
			update.Email = &request.User.Email
		case "avatar_url":
			// Validate avatar MIME type to prevent XSS during upload
			if request.User.AvatarUrl != "" {
				imageType, _, err := extractImageInfo(request.User.AvatarUrl)
				if err != nil {
					return nil, status.Errorf(codes.InvalidArgument, "invalid avatar format: %v", err)
				}
				// Only allow safe image formats for avatars
				allowedAvatarTypes := map[string]bool{
					"image/png":  true,
					"image/jpeg": true,
					"image/jpg":  true,
					"image/gif":  true,
					"image/webp": true,
				}
				if !allowedAvatarTypes[imageType] {
					return nil, status.Errorf(codes.InvalidArgument, "invalid avatar image type: %s. Only PNG, JPEG, GIF, and WebP are allowed", imageType)
				}
			}
			update.AvatarURL = &request.User.AvatarUrl
		case "description":
			update.Description = &request.User.Description
		case "role":
			// Only allow admin to update role.
			if currentUser.Role != store.RoleAdmin && currentUser.Role != store.RoleHost {
				return nil, status.Errorf(codes.PermissionDenied, "permission denied")
			}
			role := convertUserRoleToStore(request.User.Role)
			update.Role = &role
		case "password":
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(request.User.Password), bcrypt.DefaultCost)
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "failed to generate password hash").SetInternal(err)
			}
			passwordHashStr := string(passwordHash)
			update.PasswordHash = &passwordHashStr
		case "state":
			rowStatus := convertStateToStore(request.User.State)
			update.RowStatus = &rowStatus
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid update path: %s", field)
		}
	}

	updatedUser, err := s.Store.UpdateUser(ctx, update)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %v", err)
	}

	return convertUserFromStore(updatedUser), nil
}

func (s *UserService) DeleteUser(ctx context.Context, request *v1pb.DeleteUserRequest) (*emptypb.Empty, error) {
	userID, err := ExtractUserIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user name: %v", err)
	}
	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if currentUser.ID != userID && currentUser.Role != store.RoleAdmin && currentUser.Role != store.RoleHost {
		return nil, status.Errorf(codes.PermissionDenied, "permission denied")
	}

	user, err := s.Store.GetUser(ctx, &store.FindUser{ID: &userID})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if user == nil {
		return nil, status.Errorf(codes.NotFound, "user not found")
	}

	if err := s.Store.DeleteUser(ctx, &store.DeleteUser{
		ID: user.ID,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// extractUsernameFromFilter extracts username from the filter string using CEL.
// Supported filter format: "username == 'steven'"
// Returns the username value and an error if the filter format is invalid.
func extractUsernameFromFilter(filterStr string) (string, error) {
	filterStr = strings.TrimSpace(filterStr)
	if filterStr == "" {
		return "", nil
	}

	// Create CEL environment with username variable
	env, err := cel.NewEnv(
		cel.Variable("username", cel.StringType),
	)
	if err != nil {
		return "", errors.Wrap(err, "failed to create CEL environment")
	}

	// Parse and check the filter expression
	celAST, issues := env.Compile(filterStr)
	if issues != nil && issues.Err() != nil {
		return "", errors.Wrapf(issues.Err(), "invalid filter expression: %s", filterStr)
	}

	// Extract username from the AST
	username, err := extractUsernameFromAST(celAST.NativeRep().Expr())
	if err != nil {
		return "", err
	}

	return username, nil
}

// extractUsernameFromAST extracts the username value from a CEL AST expression.
func extractUsernameFromAST(expr ast.Expr) (string, error) {
	if expr == nil {
		return "", errors.New("empty expression")
	}

	// Check if this is a call expression (for ==, !=, etc.)
	if expr.Kind() != ast.CallKind {
		return "", errors.New("filter must be a comparison expression (e.g., username == 'value')")
	}

	call := expr.AsCall()

	// We only support == operator
	if call.FunctionName() != "_==_" {
		return "", errors.Errorf("unsupported operator: %s (only '==' is supported)", call.FunctionName())
	}

	// The call should have exactly 2 arguments
	args := call.Args()
	if len(args) != 2 {
		return "", errors.New("invalid comparison expression")
	}

	// Try to extract username from either left or right side
	if username, ok := extractUsernameFromComparison(args[0], args[1]); ok {
		return username, nil
	}
	if username, ok := extractUsernameFromComparison(args[1], args[0]); ok {
		return username, nil
	}

	return "", errors.New("filter must compare 'username' field with a string constant")
}

// extractUsernameFromComparison tries to extract username value if left is 'username' ident and right is a string constant.
func extractUsernameFromComparison(left, right ast.Expr) (string, bool) {
	// Check if left side is 'username' identifier
	if left.Kind() != ast.IdentKind {
		return "", false
	}
	ident := left.AsIdent()
	if ident != "username" {
		return "", false
	}

	// Right side should be a constant string
	if right.Kind() != ast.LiteralKind {
		return "", false
	}
	literal := right.AsLiteral()

	// literal is a ref.Val, we need to get the Go value
	str, ok := literal.Value().(string)
	if !ok || str == "" {
		return "", false
	}

	return str, true
}
