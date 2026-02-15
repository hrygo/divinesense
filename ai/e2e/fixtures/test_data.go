//go:build e2e_manual
// +build e2e_manual

package fixtures

import (
	"time"

	"github.com/hrygo/divinesense/store"
)

// Fixed time anchor for reproducible tests
var (
	testBaseTime = time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)
	testNow     = testBaseTime.Unix()
)

// TestUser 测试用户数据
var TestUser = &store.User{
	ID:        1,
	Username:  "test_user",
	Nickname:  "Test User",
	Email:     "test@example.com",
	Role:      store.RoleUser,
	CreatedTs: testNow,
	UpdatedTs: testNow,
	RowStatus: store.Normal,
}

// TestMemos 测试笔记数据
var TestMemos = []*store.Memo{
	{
		ID:         1,
		UID:        "memo_test_001",
		CreatorID:  1,
		Content:    "Go 语言学习笔记",
		Visibility: store.Private,
		Pinned:     false,
		CreatedTs:  testNow,
		UpdatedTs:  testNow,
		RowStatus:  store.Normal,
	},
	{
		ID:         2,
		UID:        "memo_test_002",
		CreatorID:  1,
		Content:    "会议纪要：Q1 规划",
		Visibility: store.Private,
		Pinned:     true,
		CreatedTs:  testNow,
		UpdatedTs:  testNow,
		RowStatus:  store.Normal,
	},
}

// TestSchedules 测试日程数据
// 注意: Schedule 使用 int64 时间戳 (StartTs/EndTs)
var TestSchedules = []*store.Schedule{
	{
		ID:         1,
		UID:        "schedule_test_001",
		CreatorID:  1,
		Title:      "团队周会",
		StartTs:    testBaseTime.Add(24 * time.Hour).Unix(),
		EndTs:      pointerInt64(testBaseTime.Add(25 * time.Hour).Unix()),
		AllDay:     false,
		Timezone:   "Asia/Shanghai",
		RowStatus:  store.Normal,
		CreatedTs:  testNow,
		UpdatedTs:  testNow,
	},
	{
		ID:         2,
		UID:        "schedule_test_002",
		CreatorID:  1,
		Title:      "项目评审",
		StartTs:    testBaseTime.Add(48 * time.Hour).Unix(),
		EndTs:      pointerInt64(testBaseTime.Add(50 * time.Hour).Unix()),
		AllDay:     false,
		Timezone:   "Asia/Shanghai",
		RowStatus:  store.Normal,
		CreatedTs:  testNow,
		UpdatedTs:  testNow,
	},
}

// pointerInt64 returns a pointer to the given int64 value.
func pointerInt64(v int64) *int64 {
	return &v
}
