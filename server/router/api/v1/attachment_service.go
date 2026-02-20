package v1

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lithammer/shortuuid/v4"
	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hrygo/divinesense/internal/profile"
	"github.com/hrygo/divinesense/internal/util"
	"github.com/hrygo/divinesense/plugin/filter"
	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	storepb "github.com/hrygo/divinesense/proto/gen/store"
	"github.com/hrygo/divinesense/store"
)

type AttachmentService struct {
	v1pb.UnimplementedAttachmentServiceServer
	Store              *store.Store
	Profile            *profile.Profile
	thumbnailSemaphore *semaphore.Weighted
}

const (
	// The upload memory buffer is 32 MiB.
	// It should be kept low, so RAM usage doesn't get out of control.
	// This is unrelated to maximum upload size limit, which is now set through system setting.
	MaxUploadBufferSizeBytes = 32 << 20
	MebiByte                 = 1024 * 1024
	// ThumbnailCacheFolder is the folder name where the thumbnail images are stored.
	ThumbnailCacheFolder = ".thumbnail_cache"
)

var SupportedThumbnailMimeTypes = []string{
	"image/png",
	"image/jpeg",
}

func (s *AttachmentService) CreateAttachment(ctx context.Context, request *v1pb.CreateAttachmentRequest) (*v1pb.Attachment, error) {
	user, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		slog.Error("failed to get current user", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	// Validate required fields
	if request.Attachment == nil {
		return nil, status.Errorf(codes.InvalidArgument, "attachment is required")
	}
	if request.Attachment.Filename == "" {
		return nil, status.Errorf(codes.InvalidArgument, "filename is required")
	}
	if !validateFilename(request.Attachment.Filename) {
		return nil, status.Errorf(codes.InvalidArgument, "filename contains invalid characters or format")
	}
	if request.Attachment.Type == "" {
		ext := filepath.Ext(request.Attachment.Filename)
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = http.DetectContentType(request.Attachment.Content)
		}
		// ParseMediaType to strip parameters
		mediaType, _, err := mime.ParseMediaType(mimeType)
		if err == nil {
			request.Attachment.Type = mediaType
		}
	}
	if request.Attachment.Type == "" {
		request.Attachment.Type = "application/octet-stream"
	}
	if !isValidMimeType(request.Attachment.Type) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid MIME type format")
	}

	// Use provided attachment_id or generate a new one
	attachmentUID := request.AttachmentId
	if attachmentUID == "" {
		attachmentUID = shortuuid.New()
	}

	create := &store.Attachment{
		UID:       attachmentUID,
		CreatorID: user.ID,
		Filename:  request.Attachment.Filename,
		Type:      request.Attachment.Type,
	}

	instanceStorageSetting, err := s.Store.GetInstanceStorageSetting(ctx)
	if err != nil {
		slog.Error("failed to get instance storage setting", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get instance storage setting")
	}
	uploadSizeLimit := int(instanceStorageSetting.UploadSizeLimitMb) * MebiByte
	if uploadSizeLimit == 0 {
		uploadSizeLimit = MaxUploadBufferSizeBytes
	}
	if binary.Size(request.Attachment.Content) > uploadSizeLimit {
		return nil, status.Errorf(codes.InvalidArgument, "file size exceeds the limit")
	}
	create.Size = int64(binary.Size(request.Attachment.Content))
	create.Blob = request.Attachment.Content

	if err := SaveAttachmentBlob(ctx, s.Profile, s.Store, create); err != nil {
		slog.Error("failed to save attachment blob", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to save attachment blob")
	}

	if request.Attachment.Memo != nil {
		memoUID, err := ExtractMemoUIDFromName(*request.Attachment.Memo)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid memo name")
		}
		memo, err := s.Store.GetMemo(ctx, &store.FindMemo{UID: &memoUID})
		if err != nil {
			slog.Error("failed to find memo", "error", err)
			return nil, status.Errorf(codes.Internal, "failed to find memo")
		}
		if memo == nil {
			return nil, status.Errorf(codes.NotFound, "memo not found")
		}
		create.MemoID = &memo.ID
	}
	attachment, err := s.Store.CreateAttachment(ctx, create)
	if err != nil {
		slog.Error("failed to create attachment", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create attachment")
	}

	return convertAttachmentFromStore(attachment), nil
}

func (s *AttachmentService) ListAttachments(ctx context.Context, request *v1pb.ListAttachmentsRequest) (*v1pb.ListAttachmentsResponse, error) {
	user, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		slog.Error("failed to get current user", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}

	// Set default page size
	pageSize := int(request.PageSize)
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	// Parse page token for offset
	offset := 0
	if request.PageToken != "" {
		// Simple implementation: page token is the offset as string
		// In production, you might want to use encrypted tokens
		if parsed, err := fmt.Sscanf(request.PageToken, "%d", &offset); err != nil || parsed != 1 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page token")
		}
	}

	findAttachment := &store.FindAttachment{
		CreatorID: &user.ID,
		Limit:     &pageSize,
		Offset:    &offset,
	}

	// Parse filter if provided
	if request.Filter != "" {
		if err := s.validateAttachmentFilter(ctx, request.Filter); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
		}
		findAttachment.Filters = append(findAttachment.Filters, request.Filter)
	}

	attachments, err := s.Store.ListAttachments(ctx, findAttachment)
	if err != nil {
		slog.Error("failed to list attachments", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to list attachments")
	}

	response := &v1pb.ListAttachmentsResponse{}

	for _, attachment := range attachments {
		response.Attachments = append(response.Attachments, convertAttachmentFromStore(attachment))
	}

	// For simplicity, set total size to the number of returned attachments.
	// In a full implementation, you'd want a separate count query
	response.TotalSize = int32(len(response.Attachments))

	// Set next page token if we got the full page size (indicating there might be more)
	if len(attachments) == pageSize {
		response.NextPageToken = fmt.Sprintf("%d", offset+pageSize)
	}

	return response, nil
}

func (s *AttachmentService) GetAttachment(ctx context.Context, request *v1pb.GetAttachmentRequest) (*v1pb.Attachment, error) {
	attachmentUID, err := ExtractAttachmentUIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid attachment id")
	}
	attachment, err := s.Store.GetAttachment(ctx, &store.FindAttachment{UID: &attachmentUID})
	if err != nil {
		slog.Error("failed to get attachment", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get attachment")
	}
	if attachment == nil {
		return nil, status.Errorf(codes.NotFound, "attachment not found")
	}
	return convertAttachmentFromStore(attachment), nil
}

func (s *AttachmentService) UpdateAttachment(ctx context.Context, request *v1pb.UpdateAttachmentRequest) (*v1pb.Attachment, error) {
	attachmentUID, err := ExtractAttachmentUIDFromName(request.Attachment.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid attachment id")
	}
	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update mask is required")
	}
	attachment, err := s.Store.GetAttachment(ctx, &store.FindAttachment{UID: &attachmentUID})
	if err != nil {
		slog.Error("failed to get attachment", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get attachment")
	}

	currentTs := time.Now().Unix()
	update := &store.UpdateAttachment{
		ID:        attachment.ID,
		UpdatedTs: &currentTs,
	}
	for _, field := range request.UpdateMask.Paths {
		if field == "filename" {
			if !validateFilename(request.Attachment.Filename) {
				return nil, status.Errorf(codes.InvalidArgument, "filename contains invalid characters or format")
			}
			update.Filename = &request.Attachment.Filename
		}
	}

	if err := s.Store.UpdateAttachment(ctx, update); err != nil {
		slog.Error("failed to update attachment", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to update attachment")
	}
	return s.GetAttachment(ctx, &v1pb.GetAttachmentRequest{
		Name: request.Attachment.Name,
	})
}

func (s *AttachmentService) DeleteAttachment(ctx context.Context, request *v1pb.DeleteAttachmentRequest) (*emptypb.Empty, error) {
	attachmentUID, err := ExtractAttachmentUIDFromName(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid attachment id")
	}
	user, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		slog.Error("failed to get current user", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get current user")
	}
	if user == nil {
		return nil, status.Errorf(codes.Unauthenticated, "user not authenticated")
	}
	attachment, err := s.Store.GetAttachment(ctx, &store.FindAttachment{
		UID:       &attachmentUID,
		CreatorID: &user.ID,
	})
	if err != nil {
		slog.Error("failed to find attachment", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to find attachment")
	}
	if attachment == nil {
		return nil, status.Errorf(codes.NotFound, "attachment not found")
	}
	// Delete the attachment from the database.
	if err := s.Store.DeleteAttachment(ctx, &store.DeleteAttachment{
		ID: attachment.ID,
	}); err != nil {
		slog.Error("failed to delete attachment", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to delete attachment")
	}
	return &emptypb.Empty{}, nil
}

func convertAttachmentFromStore(attachment *store.Attachment) *v1pb.Attachment {
	attachmentMessage := &v1pb.Attachment{
		Name:       fmt.Sprintf("%s%s", AttachmentNamePrefix, attachment.UID),
		CreateTime: timestamppb.New(time.Unix(attachment.CreatedTs, 0)),
		Filename:   attachment.Filename,
		Type:       attachment.Type,
		Size:       attachment.Size,
	}
	if attachment.MemoUID != nil && *attachment.MemoUID != "" {
		memoName := fmt.Sprintf("%s%s", MemoNamePrefix, *attachment.MemoUID)
		attachmentMessage.Memo = &memoName
	}
	if attachment.StorageType == storepb.AttachmentStorageType_EXTERNAL {
		attachmentMessage.ExternalLink = attachment.Reference
	}

	return attachmentMessage
}

// SaveAttachmentBlob save the blob of attachment based on the storage config.
// For personal assistant, always uses local storage.
func SaveAttachmentBlob(ctx context.Context, profile *profile.Profile, stores *store.Store, create *store.Attachment) error {
	instanceStorageSetting, err := stores.GetInstanceStorageSetting(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to find instance storage setting")
	}

	// Always use local storage for personal assistant
	filepathTemplate := "assets/{timestamp}_{filename}"
	if instanceStorageSetting.FilepathTemplate != "" {
		filepathTemplate = instanceStorageSetting.FilepathTemplate
	}

	internalPath := filepathTemplate
	if !strings.Contains(internalPath, "{filename}") {
		internalPath = filepath.Join(internalPath, "{filename}")
	}
	internalPath = replaceFilenameWithPathTemplate(internalPath, create.Filename)
	internalPath = filepath.ToSlash(internalPath)

	// Ensure the directory exists.
	osPath := filepath.FromSlash(internalPath)
	if !filepath.IsAbs(osPath) {
		osPath = filepath.Join(profile.Data, osPath)
	}
	dir := filepath.Dir(osPath)
	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return errors.Wrap(err, "Failed to create directory")
	}

	// Write the blob to the file.
	if err := os.WriteFile(osPath, create.Blob, 0644); err != nil {
		return errors.Wrap(err, "Failed to write file")
	}
	create.Reference = internalPath
	create.Blob = nil
	create.StorageType = storepb.AttachmentStorageType_LOCAL

	return nil
}

func (s *AttachmentService) GetAttachmentBlob(attachment *store.Attachment) ([]byte, error) {
	// For local storage, read the file from the local disk.
	if attachment.StorageType == storepb.AttachmentStorageType_LOCAL {
		attachmentPath := filepath.FromSlash(attachment.Reference)
		if !filepath.IsAbs(attachmentPath) {
			attachmentPath = filepath.Join(s.Profile.Data, attachmentPath)
		}

		file, err := os.Open(attachmentPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, errors.Wrap(err, "file not found")
			}
			return nil, errors.Wrap(err, "failed to open the file")
		}
		defer file.Close()
		blob, err := io.ReadAll(file)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read the file")
		}
		return blob, nil
	}
	// For database storage, return the blob from the database.
	return attachment.Blob, nil
}

var fileKeyPattern = regexp.MustCompile(`\{[a-z]{1,9}\}`)

func replaceFilenameWithPathTemplate(path, filename string) string {
	t := time.Now()
	path = fileKeyPattern.ReplaceAllStringFunc(path, func(s string) string {
		switch s {
		case "{filename}":
			return filename
		case "{timestamp}":
			return fmt.Sprintf("%d", t.Unix())
		case "{year}":
			return fmt.Sprintf("%d", t.Year())
		case "{month}":
			return fmt.Sprintf("%02d", t.Month())
		case "{day}":
			return fmt.Sprintf("%02d", t.Day())
		case "{hour}":
			return fmt.Sprintf("%02d", t.Hour())
		case "{minute}":
			return fmt.Sprintf("%02d", t.Minute())
		case "{second}":
			return fmt.Sprintf("%02d", t.Second())
		case "{uuid}":
			return util.GenUUID()
		default:
			return s
		}
	})
	return path
}

func validateFilename(filename string) bool {
	// Reject path traversal attempts and make sure no additional directories are created
	if !filepath.IsLocal(filename) || strings.ContainsAny(filename, "/\\") {
		return false
	}

	// Reject filenames starting or ending with spaces or periods
	if strings.HasPrefix(filename, " ") || strings.HasSuffix(filename, " ") ||
		strings.HasPrefix(filename, ".") || strings.HasSuffix(filename, ".") {
		return false
	}

	return true
}

func isValidMimeType(mimeType string) bool {
	// Reject empty or excessively long MIME types
	if mimeType == "" || len(mimeType) > 255 {
		return false
	}

	// MIME type must match the pattern: type/subtype
	// Allow common characters in MIME types per RFC 2045
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9!#$&^_.+-]{0,126}/[a-zA-Z0-9][a-zA-Z0-9!#$&^_.+-]{0,126}$`, mimeType)
	return matched
}

func (s *AttachmentService) validateAttachmentFilter(ctx context.Context, filterStr string) error {
	if filterStr == "" {
		return errors.New("filter cannot be empty")
	}

	engine, err := filter.DefaultAttachmentEngine()
	if err != nil {
		return err
	}

	var dialect filter.DialectName
	switch s.Profile.Driver {
	case "postgres":
		dialect = filter.DialectPostgres
	default:
		dialect = filter.DialectSQLite
	}

	if _, err := engine.CompileToStatement(ctx, filterStr, filter.RenderOptions{Dialect: dialect}); err != nil {
		return errors.Wrap(err, "failed to compile filter")
	}
	return nil
}
