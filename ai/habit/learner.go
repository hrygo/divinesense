// Package habit provides user habit learning and analysis for AI agents.
package habit

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/hrygo/divinesense/ai/services/memory"
)

// HabitLearner runs periodic habit analysis for all users.
type HabitLearner struct {
	analyzer      HabitAnalyzer
	memoryService memory.MemoryService
	config        *AnalysisConfig
	stopChan      chan struct{}
	runHour       int
	mu            sync.Mutex
	running       bool
}

// NewHabitLearner creates a new HabitLearner.
func NewHabitLearner(analyzer HabitAnalyzer, memSvc memory.MemoryService, config *AnalysisConfig) *HabitLearner {
	if config == nil {
		config = DefaultAnalysisConfig()
	}
	return &HabitLearner{
		analyzer:      analyzer,
		memoryService: memSvc,
		config:        config,
		runHour:       2, // Default: 2 AM
	}
}

// WithRunHour sets the hour of day to run analysis.
func (l *HabitLearner) WithRunHour(hour int) *HabitLearner {
	if hour >= 0 && hour < 24 {
		l.runHour = hour
	}
	return l
}

// Start begins the periodic habit analysis.
func (l *HabitLearner) Start(ctx context.Context) error {
	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return nil
	}
	l.running = true
	l.stopChan = make(chan struct{})
	l.mu.Unlock()

	// Run immediately on start
	go l.runAnalysis(ctx)

	// Schedule daily runs
	go l.scheduleDaily(ctx)

	return nil
}

// Stop stops the periodic habit analysis.
func (l *HabitLearner) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.running {
		return
	}
	l.running = false
	close(l.stopChan)
}

// scheduleDaily runs analysis at the configured hour each day.
func (l *HabitLearner) scheduleDaily(ctx context.Context) {
	for {
		// Calculate time until next run
		now := time.Now()
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), l.runHour, 0, 0, 0, now.Location())
		if now.After(nextRun) {
			nextRun = nextRun.Add(24 * time.Hour)
		}
		sleepDuration := time.Until(nextRun)

		select {
		case <-ctx.Done():
			return
		case <-l.stopChan:
			return
		case <-time.After(sleepDuration):
			l.runAnalysis(ctx)
		}
	}
}

// runAnalysis analyzes habits for all active users.
func (l *HabitLearner) runAnalysis(ctx context.Context) {
	slog.Info("starting habit analysis")
	startTime := time.Now()

	// Get active users from recent episodes
	userIDs, err := l.getActiveUsers(ctx)
	if err != nil {
		slog.Error("failed to get active users", "error", err)
		return
	}

	if len(userIDs) == 0 {
		slog.Info("no active users found for habit analysis")
		return
	}

	successCount := 0
	errorCount := 0

	for _, userID := range userIDs {
		select {
		case <-ctx.Done():
			slog.Warn("habit analysis interrupted", "processed", successCount)
			return
		default:
		}

		habits, err := l.analyzer.Analyze(ctx, userID)
		if err != nil {
			slog.Error("failed to analyze habits", "user_id", userID, "error", err)
			errorCount++
			continue
		}

		// Get existing preferences and merge with learned habits
		existingPrefs, err := l.memoryService.GetPreferences(ctx, userID)
		if err != nil {
			slog.Warn("failed to get existing preferences, will create new", "user_id", userID, "error", err)
			existingPrefs = nil
		}

		prefs := mergeHabitsToPreferences(habits, existingPrefs)
		if err := l.memoryService.UpdatePreferences(ctx, userID, prefs); err != nil {
			slog.Error("failed to update preferences", "user_id", userID, "error", err)
			errorCount++
			continue
		}

		successCount++
	}

	duration := time.Since(startTime)
	slog.Info("habit analysis completed",
		"users_processed", successCount,
		"errors", errorCount,
		"duration_ms", duration.Milliseconds())
}

// getActiveUsers returns user IDs with recent activity.
func (l *HabitLearner) getActiveUsers(ctx context.Context) ([]int32, error) {
	return l.memoryService.ListActiveUserIDs(ctx, l.config.LookbackDays)
}

// RunOnce runs habit analysis for a specific user immediately.
// Useful for testing or manual triggers.
func (l *HabitLearner) RunOnce(ctx context.Context, userID int32) (*UserHabits, error) {
	habits, err := l.analyzer.Analyze(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get existing preferences and merge with learned habits
	existingPrefs, err := l.memoryService.GetPreferences(ctx, userID)
	if err != nil {
		slog.Warn("failed to get existing preferences, will create new", "user_id", userID, "error", err)
		existingPrefs = nil
	}

	prefs := mergeHabitsToPreferences(habits, existingPrefs)
	if err := l.memoryService.UpdatePreferences(ctx, userID, prefs); err != nil {
		return nil, err
	}

	return habits, nil
}

// mergeHabitsToPreferences merges learned habits into existing preferences.
// Only habit-related fields are updated; other fields (Timezone, CommunicationStyle, etc.) are preserved.
func mergeHabitsToPreferences(habits *UserHabits, existing *memory.UserPreferences) *memory.UserPreferences {
	// Start with existing preferences or create new
	var prefs *memory.UserPreferences
	if existing != nil {
		// Clone existing to avoid mutation
		prefs = &memory.UserPreferences{
			Timezone:           existing.Timezone,
			DefaultDuration:    existing.DefaultDuration,
			PreferredTimes:     existing.PreferredTimes,
			FrequentLocations:  existing.FrequentLocations,
			CommunicationStyle: existing.CommunicationStyle,
			TagPreferences:     existing.TagPreferences,
			CustomSettings:     existing.CustomSettings,
		}
	} else {
		prefs = &memory.UserPreferences{
			CustomSettings: make(map[string]any),
		}
	}

	// Merge habit-learned fields (only if we have data)
	if habits.Time != nil && len(habits.Time.PreferredTimes) > 0 {
		prefs.PreferredTimes = habits.Time.PreferredTimes
	}

	if habits.Schedule != nil {
		if habits.Schedule.DefaultDuration > 0 {
			prefs.DefaultDuration = habits.Schedule.DefaultDuration
		}
		if len(habits.Schedule.FrequentLocations) > 0 {
			prefs.FrequentLocations = habits.Schedule.FrequentLocations
		}
	}

	if habits.Search != nil && len(habits.Search.FrequentKeywords) > 0 {
		prefs.TagPreferences = habits.Search.FrequentKeywords
	}

	return prefs
}
