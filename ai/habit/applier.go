// Package habit provides user habit learning and analysis for AI agents.
package habit

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/memory"
)

// ScheduleInput represents input for schedule creation.
type ScheduleInput struct {
	StartTime          time.Time `json:"start_time"`
	Title              string    `json:"title"`
	Location           string    `json:"location"`
	SuggestedTimes     []string  `json:"suggested_times,omitempty"`
	SuggestedLocations []string  `json:"suggested_locations,omitempty"`
	Duration           int       `json:"duration"`
	SuggestedDuration  int       `json:"suggested_duration,omitempty"`
}

// HabitApplier applies learned habits to enhance user experience.
type HabitApplier struct {
	memoryService memory.MemoryService
}

// NewHabitApplier creates a new HabitApplier.
func NewHabitApplier(memSvc memory.MemoryService) *HabitApplier {
	return &HabitApplier{
		memoryService: memSvc,
	}
}

// ApplyToScheduleCreate enhances schedule creation input with learned habits.
func (a *HabitApplier) ApplyToScheduleCreate(ctx context.Context, userID int32, input *ScheduleInput) *ScheduleInput {
	prefs, err := a.memoryService.GetPreferences(ctx, userID)
	if err != nil {
		slog.Warn("failed to get preferences for schedule enhancement", "user_id", userID, "error", err)
		return input
	}
	if prefs == nil {
		return input
	}

	// Apply default duration if not specified
	if input.Duration == 0 && prefs.DefaultDuration > 0 {
		input.SuggestedDuration = prefs.DefaultDuration
	}

	// Suggest times if start time not specified
	if input.StartTime.IsZero() && len(prefs.PreferredTimes) > 0 {
		input.SuggestedTimes = prefs.PreferredTimes
	}

	// Suggest locations if not specified
	if input.Location == "" && len(prefs.FrequentLocations) > 0 {
		input.SuggestedLocations = prefs.FrequentLocations
	}

	return input
}

// InferTime infers a specific time from a vague time reference using habits.
func (a *HabitApplier) InferTime(ctx context.Context, userID int32, query string) time.Time {
	prefs, err := a.memoryService.GetPreferences(ctx, userID)
	if err != nil {
		slog.Warn("failed to get preferences for time inference", "user_id", userID, "error", err)
		return time.Time{}
	}
	if prefs == nil || len(prefs.PreferredTimes) == 0 {
		return time.Time{}
	}

	lowerQuery := strings.ToLower(query)

	// Check for period references
	if containsPeriod(lowerQuery, "上午", "早上", "morning") {
		return findTimeInPeriod(prefs.PreferredTimes, 6, 12)
	}

	if containsPeriod(lowerQuery, "下午", "afternoon") {
		return findTimeInPeriod(prefs.PreferredTimes, 12, 18)
	}

	if containsPeriod(lowerQuery, "晚上", "evening", "night") {
		return findTimeInPeriod(prefs.PreferredTimes, 18, 24)
	}

	return time.Time{}
}

// GetSuggestedDuration returns the suggested duration for a schedule.
func (a *HabitApplier) GetSuggestedDuration(ctx context.Context, userID int32) int {
	prefs, err := a.memoryService.GetPreferences(ctx, userID)
	if err != nil {
		slog.Warn("failed to get preferences for duration suggestion", "user_id", userID, "error", err)
		return 60 // Default 1 hour
	}
	if prefs == nil || prefs.DefaultDuration == 0 {
		return 60 // Default 1 hour
	}
	return prefs.DefaultDuration
}

// GetSuggestedLocations returns frequently used locations.
func (a *HabitApplier) GetSuggestedLocations(ctx context.Context, userID int32) []string {
	prefs, err := a.memoryService.GetPreferences(ctx, userID)
	if err != nil {
		slog.Warn("failed to get preferences for location suggestions", "user_id", userID, "error", err)
		return nil
	}
	if prefs == nil {
		return nil
	}
	return prefs.FrequentLocations
}

// GetFrequentKeywords returns frequently used search keywords.
func (a *HabitApplier) GetFrequentKeywords(ctx context.Context, userID int32) []string {
	prefs, err := a.memoryService.GetPreferences(ctx, userID)
	if err != nil {
		slog.Warn("failed to get preferences for keyword suggestions", "user_id", userID, "error", err)
		return nil
	}
	if prefs == nil {
		return nil
	}
	return prefs.TagPreferences
}

// EnhanceSearchQuery suggests related keywords based on habits.
func (a *HabitApplier) EnhanceSearchQuery(ctx context.Context, userID int32, query string) []string {
	keywords := a.GetFrequentKeywords(ctx, userID)
	if len(keywords) == 0 {
		return nil
	}

	// Find related keywords
	lowerQuery := strings.ToLower(query)
	var suggestions []string

	for _, kw := range keywords {
		// Skip if keyword is already in query
		if strings.Contains(lowerQuery, strings.ToLower(kw)) {
			continue
		}

		// Simple relevance check - could be enhanced with semantic similarity
		if hasCommonChars(query, kw) {
			suggestions = append(suggestions, kw)
		}
	}

	// Limit suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// Helper functions

func containsPeriod(query string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(query, kw) {
			return true
		}
	}
	return false
}

func findTimeInPeriod(times []string, startHour, endHour int) time.Time {
	for _, t := range times {
		parsed, err := time.Parse("15:04", t)
		if err != nil {
			continue
		}

		hour := parsed.Hour()
		if hour >= startHour && hour < endHour {
			now := time.Now()
			return time.Date(now.Year(), now.Month(), now.Day(), hour, parsed.Minute(), 0, 0, now.Location())
		}
	}
	return time.Time{}
}

func hasCommonChars(a, b string) bool {
	// Check for shared characters (supports both CJK and ASCII)
	// For ASCII: requires at least 2 consecutive chars match (reduces false positives)
	// For CJK: single char match is sufficient (each char is meaningful)
	aRunes := []rune(a)
	bRunes := []rune(b)

	for i, ar := range aRunes {
		for j, br := range bRunes {
			if ar == br {
				// Non-ASCII (CJK): single char match is meaningful
				if ar > 127 {
					return true
				}
				// ASCII: check for 2+ consecutive chars (e.g., "meet" in "meeting")
				if i+1 < len(aRunes) && j+1 < len(bRunes) && aRunes[i+1] == bRunes[j+1] {
					return true
				}
			}
		}
	}
	return false
}
