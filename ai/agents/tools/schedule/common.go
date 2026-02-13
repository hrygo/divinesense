// Package schedule provides thin tool adapters for schedule operations.
// Smart LLM + Thin Tool + Rich Service pattern.
package schedule

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const DefaultTimezone = "Asia/Shanghai"

// timezoneCache caches parsed timezone locations for performance.
var timezoneCache = struct {
	locations map[string]*time.Location
	mu        sync.RWMutex
}{
	locations: make(map[string]*time.Location),
}

// getTimezoneLocation gets a timezone location from cache or loads it.
func getTimezoneLocation(timezone string) *time.Location {
	if timezone == "" {
		timezone = DefaultTimezone
	}

	timezoneCache.mu.RLock()
	loc, ok := timezoneCache.locations[timezone]
	timezoneCache.mu.RUnlock()

	if ok {
		return loc
	}

	timezoneCache.mu.Lock()
	defer timezoneCache.mu.Unlock()

	// Double-check after acquiring write lock
	if loc, ok := timezoneCache.locations[timezone]; ok {
		return loc
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		slog.Warn("failed to load timezone, using UTC", "timezone", timezone, "error", err)
		loc = time.UTC
	}

	timezoneCache.locations[timezone] = loc
	return loc
}

// parseISO8601 parses an ISO8601 time string and returns the time in UTC.
func parseISO8601(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}
	return time.Parse(time.RFC3339, s)
}

// formatTime formats a Unix timestamp for display.
func formatTime(ts int64, timezone string) string {
	t := time.Unix(ts, 0)
	loc := getTimezoneLocation(timezone)
	return t.In(loc).Format("2006-01-02 15:04 MST")
}

// normalizeJSONFields converts camelCase keys to snake_case for LLM compatibility.
func normalizeJSONFields(inputJSON string) string {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &raw); err != nil {
		return inputJSON
	}

	mappings := map[string]string{
		"startTime": "start_time",
		"endTime":   "end_time",
		"allDay":    "all_day",
		"minScore":  "min_score",
	}

	normalized := make(map[string]interface{})
	for key, value := range raw {
		newKey := key
		if mapped, ok := mappings[key]; ok {
			newKey = mapped
		}
		normalized[newKey] = value
	}

	result, err := json.Marshal(normalized)
	if err != nil {
		return inputJSON
	}
	return string(result)
}

// parseScheduleInput parses common schedule input fields.
func parseScheduleInput(inputJSON string) (map[string]interface{}, error) {
	normalized := normalizeJSONFields(inputJSON)
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(normalized), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return raw, nil
}

// getString gets a string value from a map with trimming.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// getBool gets a bool value from a map.
func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
