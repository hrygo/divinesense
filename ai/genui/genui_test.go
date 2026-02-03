package genui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduleCard(t *testing.T) {
	schedule := &ScheduleData{
		ID:        "test-123",
		Title:     "Team Meeting",
		StartTime: time.Now().Add(time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Duration:  60,
		Location:  "Conference Room A",
	}

	card := NewScheduleCard(schedule, CardStatusPreview)

	assert.Equal(t, ComponentScheduleCard, card.Type)
	assert.NotEmpty(t, card.ID)
	assert.Len(t, card.Actions, 3) // confirm, edit, cancel

	data, ok := card.Data.(*ScheduleCardData)
	require.True(t, ok)
	assert.Equal(t, "Team Meeting", data.Title)
	assert.Equal(t, "preview", data.Status)
}

func TestNewScheduleCardConflict(t *testing.T) {
	schedule := &ScheduleData{
		Title:     "Conflicting Meeting",
		StartTime: time.Now().Add(time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Duration:  60,
	}

	card := NewScheduleCard(schedule, CardStatusConflict)

	assert.Equal(t, ComponentScheduleCard, card.Type)
	assert.Len(t, card.Actions, 3) // reschedule, force_create, cancel

	// Check action IDs
	actionIDs := make([]string, len(card.Actions))
	for i, a := range card.Actions {
		actionIDs[i] = a.ID
	}
	assert.Contains(t, actionIDs, "reschedule")
	assert.Contains(t, actionIDs, "force_create")
}

func TestNewConfirmDialog(t *testing.T) {
	payload := map[string]string{"action": "delete", "id": "123"}
	dialog := NewConfirmDialog("确认删除", "确定要删除吗？", payload, true)

	assert.Equal(t, ComponentConfirmDialog, dialog.Type)
	assert.NotEmpty(t, dialog.ID)
	assert.Len(t, dialog.Actions, 2) // confirm, cancel

	data, ok := dialog.Data.(*ConfirmDialogData)
	require.True(t, ok)
	assert.Equal(t, "确认删除", data.Title)
	assert.True(t, data.Danger)

	// Check confirm button has danger style
	for _, a := range dialog.Actions {
		if a.ID == "confirm" {
			assert.Equal(t, "danger", a.Style)
		}
	}
}

func TestNewDeleteConfirmDialog(t *testing.T) {
	dialog := NewDeleteConfirmDialog("日程", "周会", map[string]string{"id": "123"})

	assert.Equal(t, ComponentConfirmDialog, dialog.Type)

	data, ok := dialog.Data.(*ConfirmDialogData)
	require.True(t, ok)
	assert.Contains(t, data.Message, "周会")
	assert.True(t, data.Danger)
}

func TestNewOptionsList(t *testing.T) {
	options := []OptionItem{
		{ID: "opt1", Label: "Option 1", Value: 1},
		{ID: "opt2", Label: "Option 2", Value: 2},
		{ID: "opt3", Label: "Option 3", Value: 3},
	}

	list := NewOptionsList("选择一项", options, false)

	assert.Equal(t, ComponentOptionsList, list.Type)
	assert.NotEmpty(t, list.ID)

	data, ok := list.Data.(*OptionsListData)
	require.True(t, ok)
	assert.Equal(t, "选择一项", data.Title)
	assert.Len(t, data.Options, 3)
	assert.False(t, data.MultiSelect)
}

func TestNewTimeSlotPicker(t *testing.T) {
	now := time.Now()
	slots := []time.Time{
		now.Add(time.Hour),
		now.Add(2 * time.Hour),
		now.Add(3 * time.Hour),
	}

	picker := NewTimeSlotPicker("选择时间", slots)

	assert.Equal(t, ComponentOptionsList, picker.Type)

	data, ok := picker.Data.(*OptionsListData)
	require.True(t, ok)
	assert.Len(t, data.Options, 3)
}

func TestNewDurationPicker(t *testing.T) {
	picker := NewDurationPicker()

	assert.Equal(t, ComponentOptionsList, picker.Type)

	data, ok := picker.Data.(*OptionsListData)
	require.True(t, ok)
	assert.Greater(t, len(data.Options), 5)

	// Check that 1 hour is selected by default
	var selected bool
	for _, opt := range data.Options {
		if opt.ID == "1hour" && opt.Selected {
			selected = true
			break
		}
	}
	assert.True(t, selected, "1 hour should be selected by default")
}

func TestNewTimePicker(t *testing.T) {
	defaultTime := time.Now().Add(24 * time.Hour)
	picker := NewTimePicker("选择日期时间", defaultTime)

	assert.Equal(t, ComponentTimePicker, picker.Type)
	assert.NotEmpty(t, picker.ID)
	assert.Len(t, picker.Actions, 2) // submit, cancel

	data, ok := picker.Data.(*TimePickerData)
	require.True(t, ok)
	assert.Equal(t, "选择日期时间", data.Label)
	assert.True(t, data.ShowTime)
	assert.Equal(t, defaultTime.Unix(), data.DefaultDate)
}

func TestNewDatePicker(t *testing.T) {
	defaultDate := time.Now().Add(24 * time.Hour)
	picker := NewDatePicker("选择日期", defaultDate)

	assert.Equal(t, ComponentTimePicker, picker.Type)

	data, ok := picker.Data.(*TimePickerData)
	require.True(t, ok)
	assert.False(t, data.ShowTime)
}

func TestUIGenerator_GenerateSchedulePreview(t *testing.T) {
	gen := NewUIGenerator()
	schedule := &ScheduleData{
		Title:     "Project Review",
		StartTime: time.Now().Add(time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Duration:  60,
	}

	resp := gen.GenerateSchedulePreview(schedule)

	assert.NotEmpty(t, resp.Text)
	assert.Len(t, resp.Components, 1)
	assert.Equal(t, ComponentScheduleCard, resp.Components[0].Type)
}

func TestUIGenerator_GenerateConfirmation(t *testing.T) {
	gen := NewUIGenerator()
	resp := gen.GenerateConfirmation("确认操作", "您确定要执行此操作吗？", nil, false)

	assert.Len(t, resp.Components, 1)
	assert.Equal(t, ComponentConfirmDialog, resp.Components[0].Type)
}

func TestUIGenerator_GenerateTimeSelection(t *testing.T) {
	gen := NewUIGenerator()

	// Test with int64 timestamp
	ts := time.Now().Add(time.Hour).Unix()
	resp := gen.GenerateTimeSelection(ts)

	assert.NotEmpty(t, resp.Text)
	assert.Len(t, resp.Components, 1)
	assert.Equal(t, ComponentTimePicker, resp.Components[0].Type)

	// Test with time.Time
	resp2 := gen.GenerateTimeSelection(time.Now().Add(2 * time.Hour))
	assert.Len(t, resp2.Components, 1)
}

func TestUIGenerator_GenerateOptionSelection(t *testing.T) {
	gen := NewUIGenerator()
	options := []OptionItem{
		{ID: "a", Label: "Option A"},
		{ID: "b", Label: "Option B"},
	}

	resp := gen.GenerateOptionSelection("请选择", options)

	assert.Len(t, resp.Components, 1)
	assert.Equal(t, ComponentOptionsList, resp.Components[0].Type)
}

func TestUIGenerator_GenerateSuccess(t *testing.T) {
	gen := NewUIGenerator()
	resp := gen.GenerateSuccess("操作成功完成！")

	assert.Len(t, resp.Components, 1)
	assert.Equal(t, ComponentSuccessBanner, resp.Components[0].Type)

	data, ok := resp.Components[0].Data.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "操作成功完成！", data["message"])
}

func TestUIGenerator_GenerateError(t *testing.T) {
	gen := NewUIGenerator()
	resp := gen.GenerateError("发生了错误")

	assert.Len(t, resp.Components, 1)
	assert.Equal(t, ComponentErrorAlert, resp.Components[0].Type)
}

func TestNewProgressBar(t *testing.T) {
	bar := NewProgressBar("处理中", 50, 100)

	assert.Equal(t, ComponentProgressBar, bar.Type)

	data, ok := bar.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "处理中", data["label"])
	assert.Equal(t, 50, data["progress"])
	assert.Equal(t, 100, data["total"])
	assert.Equal(t, 50.0, data["percent"])
}

func TestNewMemoCard(t *testing.T) {
	card := NewMemoCard("memo-123", "This is a test memo", time.Now().Unix(), []string{"work", "important"})

	assert.Equal(t, ComponentMemoCard, card.Type)
	assert.NotEmpty(t, card.ID)
	assert.Len(t, card.Actions, 1)

	data, ok := card.Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "memo-123", data["id"])
	assert.Equal(t, "This is a test memo", data["content"])
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2, "generated IDs should be unique")
	assert.Len(t, id1, 8)
}

func BenchmarkNewScheduleCard(b *testing.B) {
	schedule := &ScheduleData{
		Title:     "Benchmark Meeting",
		StartTime: time.Now().Add(time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Duration:  60,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewScheduleCard(schedule, CardStatusPreview)
	}
}

func BenchmarkUIGenerator_GenerateSchedulePreview(b *testing.B) {
	gen := NewUIGenerator()
	schedule := &ScheduleData{
		Title:     "Benchmark Meeting",
		StartTime: time.Now().Add(time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Duration:  60,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.GenerateSchedulePreview(schedule)
	}
}
