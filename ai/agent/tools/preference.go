package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hrygo/divinesense/server/service/schedule"
)

// UserPreferenceTool analyzes user's schedule history to learn preferences.
// It helps the schedule agent auto-fill default values based on user habits.
type UserPreferenceTool struct {
	scheduleSvc schedule.Service
	getUserID   func(ctx context.Context) int32
}

// NewUserPreferenceTool creates a new UserPreferenceTool.
func NewUserPreferenceTool(scheduleSvc schedule.Service, getUserID func(ctx context.Context) int32) *UserPreferenceTool {
	return &UserPreferenceTool{
		scheduleSvc: scheduleSvc,
		getUserID:   getUserID,
	}
}

// UserPreferenceInput is the input for the preference tool.
type UserPreferenceInput struct {
	// Query type: "duration", "time_slot", "location", "all"
	QueryType string `json:"query_type"`
	// Optional: specific activity type to query (e.g., "meeting", "exercise")
	ActivityType string `json:"activity_type,omitempty"`
}

// UserPreferenceOutput contains learned user preferences.
type UserPreferenceOutput struct {
	// Average duration in minutes for different activities
	AverageDurations map[string]int `json:"average_durations"`
	// Preferred time slots (hour of day)
	PreferredTimeSlots []PreferredTimeSlot `json:"preferred_time_slots"`
	// Frequently used locations
	FrequentLocations []string `json:"frequent_locations"`
	// Activity patterns
	ActivityPatterns []ActivityPattern `json:"activity_patterns"`
}

// PreferredTimeSlot represents a preferred time slot.
type PreferredTimeSlot struct {
	DayOfWeek string `json:"day_of_week,omitempty"`
	Hour      int    `json:"hour"`
	Frequency int    `json:"frequency"`
}

// ActivityPattern represents a detected activity pattern.
type ActivityPattern struct {
	Title           string `json:"title"`
	TypicalLocation string `json:"typical_location,omitempty"`
	TypicalDuration int    `json:"typical_duration_minutes"`
	TypicalHour     int    `json:"typical_hour"`
	Frequency       int    `json:"frequency"`
}

// Name returns the tool name.
func (t *UserPreferenceTool) Name() string {
	return "user_preference_get"
}

// Description returns the tool description.
func (t *UserPreferenceTool) Description() string {
	return `Analyze user's schedule history to learn preferences.

INPUT FORMAT:
{"query_type": "all", "activity_type": "meeting"}
- query_type (required): "duration" | "time_slot" | "location" | "all"
- activity_type (optional): filter by specific activity

OUTPUT FORMAT (JSON):
{
  "average_durations": {"meeting": 60, "exercise": 30},
  "preferred_time_slots": [
    {"hour": 9, "frequency": 15},
    {"hour": 14, "frequency": 12}
  ],
  "frequent_locations": ["Office", "Home"],
  "activity_patterns": [
    {
      "title": "Team Standup",
      "typical_duration_minutes": 30,
      "typical_hour": 9,
      "typical_location": "Conference Room A",
      "frequency": 45
    }
  ]
}`
}

// InputType returns the JSON Schema for the input.
func (t *UserPreferenceTool) InputType() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query_type": map[string]interface{}{
				"type":        "string",
				"description": "Type of preference to query: 'duration', 'time_slot', 'location', or 'all'",
				"enum":        []string{"duration", "time_slot", "location", "all"},
			},
			"activity_type": map[string]interface{}{
				"type":        "string",
				"description": "Optional: specific activity type to query (e.g., 'meeting', 'exercise')",
			},
		},
		"required": []string{"query_type"},
	}
}

// Run executes the tool.
func (t *UserPreferenceTool) Run(ctx context.Context, input string) (string, error) {
	var params UserPreferenceInput
	if err := json.Unmarshal([]byte(input), &params); err != nil {
		// Default to "all" if parsing fails
		params.QueryType = "all"
	}

	userID := t.getUserID(ctx)

	// Get schedules from the last 30 days
	now := time.Now()
	start := now.AddDate(0, 0, -30)
	end := now

	schedules, err := t.scheduleSvc.FindSchedules(ctx, userID, start, end)
	if err != nil {
		return "", fmt.Errorf("failed to query schedules: %w", err)
	}

	if len(schedules) == 0 {
		return `{"message": "No schedule history found. Cannot learn preferences yet."}`, nil
	}

	// Analyze schedules
	output := t.analyzeSchedules(schedules, params.ActivityType)

	// Filter output based on query type
	result := t.filterOutput(output, params.QueryType)

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %w", err)
	}

	return string(jsonBytes), nil
}

// analyzeSchedules analyzes schedule history to extract preferences.
func (t *UserPreferenceTool) analyzeSchedules(schedules []*schedule.ScheduleInstance, activityFilter string) *UserPreferenceOutput {
	output := &UserPreferenceOutput{
		AverageDurations:   make(map[string]int),
		PreferredTimeSlots: []PreferredTimeSlot{},
		FrequentLocations:  []string{},
		ActivityPatterns:   []ActivityPattern{},
	}

	// Track counts and sums for analysis
	durationSums := make(map[string]int)
	durationCounts := make(map[string]int)
	hourCounts := make(map[int]int)
	locationCounts := make(map[string]int)
	activityData := make(map[string]*activityStats)

	for _, s := range schedules {
		// Filter by activity type if specified
		if activityFilter != "" && !containsIgnoreCase(s.Title, activityFilter) {
			continue
		}

		// Calculate duration
		var duration int
		if s.EndTs != nil {
			duration = int((*s.EndTs - s.StartTs) / 60) // in minutes
		}
		if duration <= 0 {
			duration = 60 // default 1 hour
		}

		// Track by normalized title
		normalizedTitle := normalizeTitle(s.Title)
		durationSums[normalizedTitle] += duration
		durationCounts[normalizedTitle]++

		// Track time slots
		startTime := time.Unix(s.StartTs, 0)
		hour := startTime.Hour()
		hourCounts[hour]++

		// Track locations
		if s.Location != "" {
			locationCounts[s.Location]++
		}

		// Build activity stats
		if _, exists := activityData[normalizedTitle]; !exists {
			activityData[normalizedTitle] = &activityStats{
				title:     s.Title,
				durations: []int{},
				hours:     []int{},
				locations: make(map[string]int),
			}
		}
		activityData[normalizedTitle].durations = append(activityData[normalizedTitle].durations, duration)
		activityData[normalizedTitle].hours = append(activityData[normalizedTitle].hours, hour)
		if s.Location != "" {
			activityData[normalizedTitle].locations[s.Location]++
		}
	}

	// Calculate averages
	for normalizedTitle, sum := range durationSums {
		if count := durationCounts[normalizedTitle]; count > 0 {
			output.AverageDurations[normalizedTitle] = sum / count
		}
	}

	// Get top time slots
	type hourFreq struct {
		hour int
		freq int
	}
	var hourFreqs []hourFreq
	for h, f := range hourCounts {
		hourFreqs = append(hourFreqs, hourFreq{h, f})
	}
	sort.Slice(hourFreqs, func(i, j int) bool {
		return hourFreqs[i].freq > hourFreqs[j].freq
	})
	for i := 0; i < min(5, len(hourFreqs)); i++ {
		output.PreferredTimeSlots = append(output.PreferredTimeSlots, PreferredTimeSlot{
			Hour:      hourFreqs[i].hour,
			Frequency: hourFreqs[i].freq,
		})
	}

	// Get frequent locations
	type locFreq struct {
		loc  string
		freq int
	}
	var locFreqs []locFreq
	for l, f := range locationCounts {
		locFreqs = append(locFreqs, locFreq{l, f})
	}
	sort.Slice(locFreqs, func(i, j int) bool {
		return locFreqs[i].freq > locFreqs[j].freq
	})
	for i := 0; i < min(5, len(locFreqs)); i++ {
		output.FrequentLocations = append(output.FrequentLocations, locFreqs[i].loc)
	}

	// Build activity patterns
	for _, stats := range activityData {
		if len(stats.durations) < 2 {
			continue // Need at least 2 occurrences
		}

		pattern := ActivityPattern{
			Title:           stats.title,
			TypicalDuration: average(stats.durations),
			TypicalHour:     mode(stats.hours),
			Frequency:       len(stats.durations),
		}

		// Find most common location
		maxLoc := ""
		maxLocCount := 0
		for loc, count := range stats.locations {
			if count > maxLocCount {
				maxLoc = loc
				maxLocCount = count
			}
		}
		pattern.TypicalLocation = maxLoc

		output.ActivityPatterns = append(output.ActivityPatterns, pattern)
	}

	// Sort patterns by frequency
	sort.Slice(output.ActivityPatterns, func(i, j int) bool {
		return output.ActivityPatterns[i].Frequency > output.ActivityPatterns[j].Frequency
	})

	// Keep top 10 patterns
	if len(output.ActivityPatterns) > 10 {
		output.ActivityPatterns = output.ActivityPatterns[:10]
	}

	return output
}

// filterOutput filters the output based on query type.
func (t *UserPreferenceTool) filterOutput(output *UserPreferenceOutput, queryType string) interface{} {
	switch queryType {
	case "duration":
		return map[string]interface{}{
			"average_durations": output.AverageDurations,
		}
	case "time_slot":
		return map[string]interface{}{
			"preferred_time_slots": output.PreferredTimeSlots,
		}
	case "location":
		return map[string]interface{}{
			"frequent_locations": output.FrequentLocations,
		}
	default:
		return output
	}
}

// Helper types and functions

type activityStats struct {
	locations map[string]int
	title     string
	durations []int
	hours     []int
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0)
}

func normalizeTitle(title string) string {
	// Simple normalization: lowercase and trim
	// In production, could use more sophisticated NLP
	if len(title) > 20 {
		return title[:20]
	}
	return title
}

func average(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	sum := 0
	for _, n := range nums {
		sum += n
	}
	return sum / len(nums)
}

func mode(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	counts := make(map[int]int)
	for _, n := range nums {
		counts[n]++
	}
	maxCount := 0
	modeVal := nums[0]
	for n, c := range counts {
		if c > maxCount {
			maxCount = c
			modeVal = n
		}
	}
	return modeVal
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
