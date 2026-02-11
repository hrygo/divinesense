// Package habit provides user habit learning and analysis for AI agents.
package habit

import (
	"context"
	"sort"
	"time"

	"github.com/hrygo/divinesense/ai/services/memory"
)

// HabitAnalyzer analyzes user interaction history to learn habits.
type HabitAnalyzer interface {
	// Analyze analyzes user habits from historical data.
	Analyze(ctx context.Context, userID int32) (*UserHabits, error)
}

// habitAnalyzer is the default implementation of HabitAnalyzer.
type habitAnalyzer struct {
	memoryService memory.MemoryService
	config        *AnalysisConfig
}

// NewHabitAnalyzer creates a new HabitAnalyzer instance.
func NewHabitAnalyzer(memSvc memory.MemoryService, config *AnalysisConfig) HabitAnalyzer {
	if config == nil {
		config = DefaultAnalysisConfig()
	}
	return &habitAnalyzer{
		memoryService: memSvc,
		config:        config,
	}
}

// Analyze implements HabitAnalyzer.Analyze.
func (a *habitAnalyzer) Analyze(ctx context.Context, userID int32) (*UserHabits, error) {
	// Search for recent episodes
	episodes, err := a.memoryService.SearchEpisodes(ctx, userID, "", 500)
	if err != nil {
		return nil, err
	}

	// Filter to lookback window and successful outcomes
	cutoff := time.Now().AddDate(0, 0, -a.config.LookbackDays)
	var filteredEpisodes []memory.EpisodicMemory
	for _, ep := range episodes {
		if ep.Timestamp.After(cutoff) && ep.Outcome == "success" {
			filteredEpisodes = append(filteredEpisodes, ep)
		}
	}

	// Check minimum samples
	if len(filteredEpisodes) < a.config.MinSamples {
		return DefaultUserHabits(userID), nil
	}

	// Analyze each dimension
	timeHabits := a.analyzeTimeHabits(filteredEpisodes)
	scheduleHabits := a.analyzeScheduleHabits(filteredEpisodes)
	searchHabits := a.analyzeSearchHabits(filteredEpisodes)

	return &UserHabits{
		UserID:    userID,
		Time:      timeHabits,
		Schedule:  scheduleHabits,
		Search:    searchHabits,
		UpdatedAt: time.Now(),
	}, nil
}

// analyzeTimeHabits analyzes time-related habits.
func (a *habitAnalyzer) analyzeTimeHabits(episodes []memory.EpisodicMemory) *TimeHabits {
	if len(episodes) == 0 {
		return DefaultTimeHabits()
	}

	// Count hours and weekday/weekend
	hourCounts := make(map[int]int)
	weekdayCount := 0
	weekendCount := 0

	for _, ep := range episodes {
		hour := ep.Timestamp.Hour()
		hourCounts[hour]++

		weekday := ep.Timestamp.Weekday()
		if weekday >= time.Monday && weekday <= time.Friday {
			weekdayCount++
		} else {
			weekendCount++
		}
	}

	// Find top-5 active hours
	activeHours := topNHours(hourCounts, 5)

	// Infer preferred times from peaks
	preferredTimes := a.inferPreferredTimes(hourCounts)

	return &TimeHabits{
		ActiveHours:     activeHours,
		PreferredTimes:  preferredTimes,
		ReminderLeadMin: 15, // Default
		WeekdayPattern:  weekdayCount > weekendCount*2,
	}
}

// analyzeScheduleHabits analyzes schedule-related habits.
func (a *habitAnalyzer) analyzeScheduleHabits(episodes []memory.EpisodicMemory) *ScheduleHabits {
	// Filter to schedule-related episodes
	var scheduleEpisodes []memory.EpisodicMemory
	for _, ep := range episodes {
		if ep.AgentType == "schedule" {
			scheduleEpisodes = append(scheduleEpisodes, ep)
		}
	}

	if len(scheduleEpisodes) < 5 {
		return DefaultScheduleHabits()
	}

	// Analyze preferred time slots
	slotCounts := make(map[string]int)
	for _, ep := range scheduleEpisodes {
		hour := ep.Timestamp.Hour()
		slot := hourToSlot(hour)
		slotCounts[slot]++
	}

	preferredSlots := topNStrings(slotCounts, 3)

	return &ScheduleHabits{
		DefaultDuration:   60, // Default 1 hour
		PreferredSlots:    preferredSlots,
		FrequentLocations: []string{},
		TitlePatterns:     []string{},
	}
}

// analyzeSearchHabits analyzes search-related habits.
func (a *habitAnalyzer) analyzeSearchHabits(episodes []memory.EpisodicMemory) *SearchHabits {
	// Filter to memo/search-related episodes
	var searchEpisodes []memory.EpisodicMemory
	for _, ep := range episodes {
		if ep.AgentType == "memo" {
			searchEpisodes = append(searchEpisodes, ep)
		}
	}

	if len(searchEpisodes) < 5 {
		return DefaultSearchHabits()
	}

	// Extract keywords from user inputs
	keywordCounts := make(map[string]int)
	exactCount := 0
	fuzzyCount := 0

	for _, ep := range searchEpisodes {
		words := tokenize(ep.UserInput)
		for _, word := range words {
			if len(word) >= 2 { // Skip single chars
				keywordCounts[word]++
			}
		}

		// Check search mode
		if hasExactQuotes(ep.UserInput) {
			exactCount++
		} else {
			fuzzyCount++
		}
	}

	// Filter keywords by minimum frequency
	frequentKeywords := filterByFrequency(keywordCounts, a.config.MinKeywordFrequency)

	searchMode := "fuzzy"
	if exactCount > fuzzyCount {
		searchMode = "exact"
	}

	return &SearchHabits{
		FrequentKeywords: frequentKeywords,
		SearchMode:       searchMode,
		ResultPreference: "",
	}
}

// inferPreferredTimes finds peak hours and converts to time strings.
func (a *habitAnalyzer) inferPreferredTimes(hourCounts map[int]int) []string {
	if len(hourCounts) == 0 {
		return []string{"09:00", "14:00"}
	}

	// Calculate average
	total := 0
	for _, count := range hourCounts {
		total += count
	}
	avg := float64(total) / float64(len(hourCounts))
	threshold := int(avg * a.config.PeakMultiplier)

	// Find peak hours
	var peaks []int
	for hour, count := range hourCounts {
		if count >= threshold {
			peaks = append(peaks, hour)
		}
	}
	sort.Ints(peaks)

	// Convert to time strings
	var times []string
	for _, hour := range peaks {
		times = append(times, hourToTimeString(hour))
	}

	if len(times) == 0 {
		return []string{"09:00", "14:00"}
	}
	return times
}

// Helper functions

func topNHours(hourCounts map[int]int, n int) []int {
	type hourCount struct {
		hour  int
		count int
	}

	hcs := make([]hourCount, 0, len(hourCounts))
	for hour, count := range hourCounts {
		hcs = append(hcs, hourCount{hour, count})
	}

	sort.Slice(hcs, func(i, j int) bool {
		return hcs[i].count > hcs[j].count
	})

	var result []int
	for i := 0; i < n && i < len(hcs); i++ {
		result = append(result, hcs[i].hour)
	}
	sort.Ints(result)
	return result
}

func topNStrings(counts map[string]int, n int) []string {
	type strCount struct {
		str   string
		count int
	}

	scs := make([]strCount, 0, len(counts))
	for s, count := range counts {
		scs = append(scs, strCount{s, count})
	}

	sort.Slice(scs, func(i, j int) bool {
		return scs[i].count > scs[j].count
	})

	var result []string
	for i := 0; i < n && i < len(scs); i++ {
		result = append(result, scs[i].str)
	}
	return result
}

func filterByFrequency(counts map[string]int, minFreq int) []string {
	var result []string
	for s, count := range counts {
		if count >= minFreq {
			result = append(result, s)
		}
	}
	// Sort by frequency descending
	sort.Slice(result, func(i, j int) bool {
		return counts[result[i]] > counts[result[j]]
	})
	// Limit to top 10
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

func hourToSlot(hour int) string {
	switch {
	case hour >= 6 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 14:
		return "noon"
	case hour >= 14 && hour < 18:
		return "afternoon"
	case hour >= 18 && hour < 22:
		return "evening"
	default:
		return "night"
	}
}

func hourToTimeString(hour int) string {
	return time.Date(0, 1, 1, hour, 0, 0, 0, time.UTC).Format("15:04")
}

func tokenize(input string) []string {
	// Simple tokenization by common delimiters
	var words []string
	var current []rune

	for _, r := range input {
		if r == ' ' || r == ',' || r == '。' || r == '，' || r == '?' || r == '？' || r == '!' || r == '！' {
			if len(current) > 0 {
				words = append(words, string(current))
				current = nil
			}
		} else {
			current = append(current, r)
		}
	}
	if len(current) > 0 {
		words = append(words, string(current))
	}
	return words
}

func hasExactQuotes(input string) bool {
	return len(input) > 2 && ((input[0] == '"' && input[len(input)-1] == '"') ||
		(input[0] == '\'' && input[len(input)-1] == '\''))
}
