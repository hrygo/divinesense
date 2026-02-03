package schedule

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hrygo/divinesense/ai/aitime"
	"github.com/hrygo/divinesense/ai/habit"
	"github.com/hrygo/divinesense/ai/memory"
)

// mockMemoryService for testing habit applier.
type mockMemoryService struct {
	preferences *memory.UserPreferences
}

func (m *mockMemoryService) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]memory.Message, error) {
	return nil, nil
}
func (m *mockMemoryService) AddMessage(ctx context.Context, sessionID string, msg memory.Message) error {
	return nil
}
func (m *mockMemoryService) SaveEpisode(ctx context.Context, episode memory.EpisodicMemory) error {
	return nil
}
func (m *mockMemoryService) SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]memory.EpisodicMemory, error) {
	return nil, nil
}
func (m *mockMemoryService) ListActiveUserIDs(ctx context.Context, lookbackDays int) ([]int32, error) {
	return nil, nil
}
func (m *mockMemoryService) GetPreferences(ctx context.Context, userID int32) (*memory.UserPreferences, error) {
	return m.preferences, nil
}
func (m *mockMemoryService) UpdatePreferences(ctx context.Context, userID int32, prefs *memory.UserPreferences) error {
	return nil
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"明天下午3点开会", "会议"},
		{"后天早上9点面试", "面试"},
		{"今天晚上8点讨论项目", "讨论"},
		{"下周一上午10点汇报工作", "工作汇报"},
		{"明天下午meeting", "Meeting"},
		{"帮我安排明天上午的review", "Review"},
		{"今天下午3点电话会议", "电话会议"},
		{"明天约客户吃饭", "约会"},
		{"周五做ppt", "做ppt"}, // No action mapping, use cleaned text
	}

	for _, tt := range tests {
		result := extractTitle(tt.input)
		if result != tt.expected {
			t.Errorf("extractTitle(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCalculateConfidence(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	tests := []struct {
		schedule *ScheduleRequest
		name     string
		minScore float64
		maxScore float64
	}{
		{
			name: "complete schedule",
			schedule: &ScheduleRequest{
				Title:     "会议",
				StartTime: future,
				Duration:  60,
			},
			minScore: 0.9,
			maxScore: 1.0,
		},
		{
			name: "missing title",
			schedule: &ScheduleRequest{
				StartTime: future,
				Duration:  60,
			},
			minScore: 0.0,
			maxScore: 0.6,
		},
		{
			name: "missing time",
			schedule: &ScheduleRequest{
				Title:    "会议",
				Duration: 60,
			},
			minScore: 0.0,
			maxScore: 0.6,
		},
		{
			name: "past time",
			schedule: &ScheduleRequest{
				Title:     "会议",
				StartTime: past,
				Duration:  60,
			},
			minScore: 0.6,
			maxScore: 0.85,
		},
		{
			name: "invalid duration",
			schedule: &ScheduleRequest{
				Title:     "会议",
				StartTime: future,
				Duration:  -10,
			},
			minScore: 0.8,
			maxScore: 0.95,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateConfidence(tt.schedule)
			if score < tt.minScore || score > tt.maxScore {
				t.Errorf("calculateConfidence() = %v, want between %v and %v", score, tt.minScore, tt.maxScore)
			}
		})
	}
}

func TestFastCreateParser_Parse(t *testing.T) {
	timeSvc := aitime.NewService("Asia/Shanghai")
	mockMem := &mockMemoryService{
		preferences: &memory.UserPreferences{
			DefaultDuration:   45,
			FrequentLocations: []string{"会议室A"},
		},
	}
	habitApplier := habit.NewHabitApplier(mockMem)

	parser := NewFastCreateParser(timeSvc, habitApplier)

	tests := []struct {
		name      string
		input     string
		wantTitle string
		canFast   bool
	}{
		{
			name:      "simple create with time and action",
			input:     "明天下午3点开会",
			canFast:   true,
			wantTitle: "会议",
		},
		{
			name:      "create with meeting keyword",
			input:     "后天早上9点面试",
			canFast:   true,
			wantTitle: "面试",
		},
		{
			name:    "query intent - should not fast create",
			input:   "今天有什么安排？",
			canFast: false,
		},
		{
			name:    "unclear intent",
			input:   "明天做点什么",
			canFast: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(context.Background(), 1, tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if result.CanFastCreate != tt.canFast {
				t.Errorf("CanFastCreate = %v, want %v (missing: %v)", result.CanFastCreate, tt.canFast, result.MissingFields)
			}

			if tt.canFast && tt.wantTitle != "" && result.Schedule.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Schedule.Title, tt.wantTitle)
			}

			// Check defaults are applied
			if tt.canFast {
				if result.Schedule.Duration == 0 {
					t.Error("Duration should have default value")
				}
				if result.Schedule.ReminderMinutes == 0 {
					t.Error("ReminderMinutes should have default value")
				}
			}
		})
	}
}

func TestFastCreateHandler_Handle(t *testing.T) {
	timeSvc := aitime.NewService("Asia/Shanghai")
	parser := NewFastCreateParser(timeSvc, nil)
	handler := NewFastCreateHandler(parser)

	tests := []struct {
		name        string
		input       string
		wantType    ResponseType
		wantActions int
	}{
		{
			name:        "fast create success",
			input:       "明天下午3点开会",
			wantType:    ResponseTypeFastCreate,
			wantActions: 3, // confirm, edit, cancel
		},
		{
			name:        "fallback on query",
			input:       "今天有什么安排？",
			wantType:    ResponseTypeFallback,
			wantActions: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := handler.Handle(context.Background(), 1, tt.input)
			if err != nil {
				t.Fatalf("Handle error: %v", err)
			}

			if resp.Type != tt.wantType {
				t.Errorf("Response type = %v, want %v", resp.Type, tt.wantType)
			}

			if tt.wantActions > 0 && len(resp.Actions) != tt.wantActions {
				t.Errorf("Actions count = %d, want %d", len(resp.Actions), tt.wantActions)
			}
		})
	}
}

func TestGeneratePreview(t *testing.T) {
	schedule := &ScheduleRequest{
		Title:           "会议",
		StartTime:       time.Date(2026, 1, 28, 15, 0, 0, 0, time.Local),
		EndTime:         time.Date(2026, 1, 28, 16, 0, 0, 0, time.Local),
		Duration:        60,
		Location:        "会议室A",
		ReminderMinutes: 15,
	}

	preview := generatePreview(schedule)

	// Check that preview contains essential info
	if preview == "" {
		t.Error("Preview should not be empty")
	}

	checks := []string{"会议", "01月28日", "15:00", "16:00", "60 分钟", "会议室A", "15 分钟"}
	for _, check := range checks {
		if !strings.Contains(preview, check) {
			t.Errorf("Preview should contain %q, got: %s", check, preview)
		}
	}
}

func TestApplyDefaults(t *testing.T) {
	mockMem := &mockMemoryService{
		preferences: &memory.UserPreferences{
			DefaultDuration:   45,
			FrequentLocations: []string{"会议室A", "咖啡厅"},
		},
	}
	habitApplier := habit.NewHabitApplier(mockMem)
	parser := NewFastCreateParser(nil, habitApplier)

	schedule := &ScheduleRequest{
		Title:     "测试会议",
		StartTime: time.Now().Add(24 * time.Hour),
	}

	parser.applyDefaults(context.Background(), 1, schedule)

	// Check duration from habit (45) or default (60)
	if schedule.Duration == 0 {
		t.Error("Duration should be set")
	}

	// Check reminder default
	if schedule.ReminderMinutes != 15 {
		t.Errorf("ReminderMinutes = %d, want 15", schedule.ReminderMinutes)
	}

	// Check end time calculated
	if schedule.EndTime.IsZero() {
		t.Error("EndTime should be calculated")
	}
}

// Benchmark tests.
func BenchmarkExtractTitle(b *testing.B) {
	input := "明天下午3点开会讨论项目进度"
	for i := 0; i < b.N; i++ {
		extractTitle(input)
	}
}

func BenchmarkFastCreateParse(b *testing.B) {
	timeSvc := aitime.NewService("Asia/Shanghai")
	parser := NewFastCreateParser(timeSvc, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser.Parse(ctx, 1, "明天下午3点开会")
	}
}
