package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1pb "github.com/hrygo/divinesense/proto/gen/api/v1"
	"github.com/hrygo/divinesense/store"
)

// memoStatsCollector accumulates statistics for memos.
type memoStatsCollector struct {
	displayTimestamps []*timestamppb.Timestamp
	tagCount          map[string]int32
	linkCount         int32
	codeCount         int32
	todoCount         int32
	undoCount         int32
	pinnedMemos       []string
	totalMemoCount    int32
}

// newMemoStatsCollector creates a new stats collector with pre-allocated slices.
func newMemoStatsCollector() *memoStatsCollector {
	return &memoStatsCollector{
		tagCount:          make(map[string]int32),
		displayTimestamps: make([]*timestamppb.Timestamp, 0, 64), // Pre-allocate for efficiency
		pinnedMemos:       make([]string, 0, 8),                  // Pre-allocate for efficiency
	}
}

// collect processes a single memo and updates statistics.
//
// Parameters:
//   - memo: The memo to collect statistics from. Must not be nil.
//   - userID: The user ID for generating pinned memo resource names.
//   - displayWithUpdate: If true, use UpdatedTs for display timestamp; otherwise use CreatedTs.
func (c *memoStatsCollector) collect(memo *store.Memo, userID int32, displayWithUpdate bool) {
	// Display timestamp
	displayTs := memo.CreatedTs
	if displayWithUpdate {
		displayTs = memo.UpdatedTs
	}
	c.displayTimestamps = append(c.displayTimestamps, timestamppb.New(time.Unix(displayTs, 0)))
	c.totalMemoCount++

	// Pinned memos
	if memo.Pinned {
		c.pinnedMemos = append(c.pinnedMemos, fmt.Sprintf("users/%d/memos/%d", userID, memo.ID))
	}

	// Early return if no payload
	payload := memo.Payload
	if payload == nil {
		return
	}

	// Count tags
	for _, tag := range payload.Tags {
		c.tagCount[tag]++
	}

	// Early return if no property
	prop := payload.Property
	if prop == nil {
		return
	}

	// Count memo types
	if prop.HasLink {
		c.linkCount++
	}
	if prop.HasCode {
		c.codeCount++
	}
	if prop.HasTaskList {
		c.todoCount++
	}
	if prop.HasIncompleteTasks {
		c.undoCount++
	}
}

// toProto converts collector to protobuf UserStats.
func (c *memoStatsCollector) toProto(userID int32) *v1pb.UserStats {
	return &v1pb.UserStats{
		Name:                  fmt.Sprintf("users/%d/stats", userID),
		MemoDisplayTimestamps: c.displayTimestamps,
		TagCount:              c.tagCount,
		PinnedMemos:           c.pinnedMemos,
		TotalMemoCount:        c.totalMemoCount,
		MemoTypeStats: &v1pb.UserStats_MemoTypeStats{
			LinkCount: c.linkCount,
			CodeCount: c.codeCount,
			TodoCount: c.todoCount,
			UndoCount: c.undoCount,
		},
	}
}

func (s *UserService) ListAllUserStats(ctx context.Context, _ *v1pb.ListAllUserStatsRequest) (*v1pb.ListAllUserStatsResponse, error) {
	instanceSetting, err := s.Store.GetInstanceMemoRelatedSetting(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get instance memo setting")
	}

	// Build memo finder with visibility filter based on current user
	normalStatus := store.Normal
	memoFind := &store.FindMemo{
		ExcludeComments: true,
		ExcludeContent:  true,
		RowStatus:       &normalStatus,
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if currentUser == nil {
		memoFind.VisibilityList = []store.Visibility{store.Public}
	} else if memoFind.CreatorID == nil {
		filter := fmt.Sprintf(`creator_id == %d || visibility in ["PUBLIC", "PROTECTED"]`, currentUser.ID)
		memoFind.Filters = append(memoFind.Filters, filter)
	} else if *memoFind.CreatorID != currentUser.ID {
		memoFind.VisibilityList = []store.Visibility{store.Public, store.Protected}
	}

	// Collect stats per user
	collectors := make(map[int32]*memoStatsCollector)
	limit := 1000
	memoFind.Limit = &limit
	offset := 0

	for {
		memoFind.Offset = &offset
		memos, err := s.Store.ListMemos(ctx, memoFind)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list memos: %v", err)
		}
		if len(memos) == 0 {
			break
		}

		for _, memo := range memos {
			if _, exists := collectors[memo.CreatorID]; !exists {
				collectors[memo.CreatorID] = newMemoStatsCollector()
			}
			collectors[memo.CreatorID].collect(memo, memo.CreatorID, instanceSetting.DisplayWithUpdateTime)
		}

		offset += limit
	}

	// Convert to response
	stats := make([]*v1pb.UserStats, 0, len(collectors))
	for userID, collector := range collectors {
		stats = append(stats, collector.toProto(userID))
	}

	return &v1pb.ListAllUserStatsResponse{Stats: stats}, nil
}

func (s *UserService) GetUserStats(ctx context.Context, req *v1pb.GetUserStatsRequest) (*v1pb.UserStats, error) {
	userID, err := ExtractUserIDFromName(req.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user name: %v", err)
	}

	instanceSetting, err := s.Store.GetInstanceMemoRelatedSetting(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get instance memo setting")
	}

	// Build memo finder with visibility filter
	normalStatus := store.Normal
	memoFind := &store.FindMemo{
		CreatorID:       &userID,
		ExcludeComments: true,
		ExcludeContent:  true,
		RowStatus:       &normalStatus,
	}

	currentUser, err := fetchCurrentUser(ctx, s.Store)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %v", err)
	}
	if currentUser == nil {
		memoFind.VisibilityList = []store.Visibility{store.Public}
	} else if currentUser.ID != userID {
		memoFind.VisibilityList = []store.Visibility{store.Public, store.Protected}
	}

	// Collect stats
	collector := newMemoStatsCollector()
	limit := 1000
	memoFind.Limit = &limit
	offset := 0

	for {
		memoFind.Offset = &offset
		memos, err := s.Store.ListMemos(ctx, memoFind)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list memos: %v", err)
		}
		if len(memos) == 0 {
			break
		}

		for _, memo := range memos {
			collector.collect(memo, userID, instanceSetting.DisplayWithUpdateTime)
		}

		offset += limit
	}

	return collector.toProto(userID), nil
}
