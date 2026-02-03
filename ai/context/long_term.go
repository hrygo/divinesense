// Package context provides context building for LLM prompts.
package context

import (
	"context"
	"fmt"
	"strings"
)

// LongTermExtractor extracts episodic memories and user preferences.
type LongTermExtractor struct {
	maxEpisodes int
}

// NewLongTermExtractor creates a new long-term memory extractor.
func NewLongTermExtractor(maxEpisodes int) *LongTermExtractor {
	if maxEpisodes <= 0 {
		maxEpisodes = 3
	}
	return &LongTermExtractor{
		maxEpisodes: maxEpisodes,
	}
}

// EpisodicProvider provides episodic memory search.
type EpisodicProvider interface {
	SearchEpisodes(ctx context.Context, userID int32, query string, limit int) ([]*EpisodicMemory, error)
}

// PreferenceProvider provides user preferences.
type PreferenceProvider interface {
	GetPreferences(ctx context.Context, userID int32) (*UserPreferences, error)
}

// LongTermContext contains extracted long-term context.
type LongTermContext struct {
	Preferences *UserPreferences
	Episodes    []*EpisodicMemory
}

// Extract extracts long-term context for the user.
func (e *LongTermExtractor) Extract(
	ctx context.Context,
	episodicProvider EpisodicProvider,
	prefProvider PreferenceProvider,
	userID int32,
	query string,
) (*LongTermContext, error) {
	result := &LongTermContext{}

	// Extract episodic memories
	if episodicProvider != nil {
		episodes, err := episodicProvider.SearchEpisodes(ctx, userID, query, e.maxEpisodes)
		if err != nil {
			// Non-fatal: continue without episodes
			episodes = nil
		}
		result.Episodes = episodes
	}

	// Extract user preferences
	if prefProvider != nil {
		prefs, err := prefProvider.GetPreferences(ctx, userID)
		if err != nil {
			// Non-fatal: use defaults
			prefs = DefaultUserPreferences()
		}
		result.Preferences = prefs
	}

	return result, nil
}

// FormatEpisodes formats episodic memories into context.
func FormatEpisodes(episodes []*EpisodicMemory) string {
	if len(episodes) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("### 相关历史\n")

	for _, ep := range episodes {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n",
			ep.Timestamp.Format("01-02 15:04"),
			ep.Summary))
	}

	return sb.String()
}

// FormatPreferences formats user preferences into context.
func FormatPreferences(prefs *UserPreferences) string {
	if prefs == nil {
		return ""
	}

	var parts []string

	if prefs.Timezone != "" {
		parts = append(parts, fmt.Sprintf("时区: %s", prefs.Timezone))
	}
	if prefs.DefaultDuration > 0 {
		parts = append(parts, fmt.Sprintf("默认会议时长: %d分钟", prefs.DefaultDuration))
	}
	if len(prefs.PreferredTimes) > 0 {
		parts = append(parts, fmt.Sprintf("偏好时间: %s", strings.Join(prefs.PreferredTimes, ", ")))
	}
	if prefs.CommunicationStyle != "" {
		parts = append(parts, fmt.Sprintf("沟通风格: %s", prefs.CommunicationStyle))
	}

	if len(parts) == 0 {
		return ""
	}

	return "### 用户偏好\n" + strings.Join(parts, " | ")
}

// DefaultUserPreferences returns default preferences.
func DefaultUserPreferences() *UserPreferences {
	return &UserPreferences{
		Timezone:           "Asia/Shanghai",
		DefaultDuration:    60,
		PreferredTimes:     []string{"09:00", "14:00"},
		CommunicationStyle: "concise",
	}
}
