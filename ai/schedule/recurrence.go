package schedule

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RecurrenceType represents the type of recurrence pattern.
type RecurrenceType string

const (
	// RecurrenceTypeDaily represents daily recurrence.
	RecurrenceTypeDaily RecurrenceType = "daily"
	// RecurrenceTypeWeekly represents weekly recurrence.
	RecurrenceTypeWeekly RecurrenceType = "weekly"
	// RecurrenceTypeMonthly represents monthly recurrence.
	RecurrenceTypeMonthly RecurrenceType = "monthly"
)

// IsValid checks if the recurrence type is valid.
func (rt RecurrenceType) IsValid() bool {
	switch rt {
	case RecurrenceTypeDaily, RecurrenceTypeWeekly, RecurrenceTypeMonthly:
		return true
	default:
		return false
	}
}

// String returns the string representation of RecurrenceType.
func (rt RecurrenceType) String() string {
	return string(rt)
}

// RecurrenceRule represents a simplified recurrence rule.
// We use a custom JSON format instead of full RFC 5545 RRULE for simplicity.
type RecurrenceRule struct {
	Type     RecurrenceType `json:"type"`
	Weekdays []int          `json:"weekdays"`
	Interval int            `json:"interval"`
	MonthDay int            `json:"month_day"`
}

// Validate checks if the recurrence rule is valid.
func (r *RecurrenceRule) Validate() error {
	if !r.Type.IsValid() {
		return fmt.Errorf("invalid recurrence type: %s", r.Type)
	}
	if r.Interval <= 0 {
		return fmt.Errorf("interval must be positive, got: %d", r.Interval)
	}

	switch r.Type {
	case RecurrenceTypeWeekly:
		if len(r.Weekdays) == 0 {
			return fmt.Errorf("weekdays required for weekly recurrence")
		}
		for _, day := range r.Weekdays {
			if day < 1 || day > 7 {
				return fmt.Errorf("invalid weekday: %d (must be 1-7)", day)
			}
		}
	case RecurrenceTypeMonthly:
		if r.MonthDay < 1 || r.MonthDay > 31 {
			return fmt.Errorf("invalid month_day: %d (must be 1-31)", r.MonthDay)
		}
	}

	return nil
}

// ParseRecurrenceRule parses a natural language recurrence pattern.
// Examples:
//   - "每天" → {Type: RecurrenceTypeDaily, Interval: 1}
//   - "每3天" → {Type: RecurrenceTypeDaily, Interval: 3}
//   - "每周一" → {Type: RecurrenceTypeWeekly, Weekdays: [1]}
//   - "每周" → {Type: RecurrenceTypeWeekly, Interval: 1}
//   - "每两周" → {Type: RecurrenceTypeDaily, Interval: 14}
//   - "每月15号" → {Type: RecurrenceTypeMonthly, MonthDay: 15}
func ParseRecurrenceRule(text string) (*RecurrenceRule, error) {
	text = strings.TrimSpace(text)

	// Daily patterns
	if matched, _ := regexp.MatchString(`^(每|每天)(\d+)?天?$`, text); matched {
		rule := &RecurrenceRule{Type: RecurrenceTypeDaily, Interval: 1}
		if parts := regexp.MustCompile(`(\d+)`).FindStringSubmatch(text); len(parts) > 1 {
			if interval := parseInt(parts[1]); interval > 0 {
				rule.Interval = interval
			}
		}
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("invalid recurrence rule: %w", err)
		}
		return rule, nil
	}

	// Weekly patterns
	if matched, _ := regexp.MatchString(`^每(\d+)?(周|星期)(一|二|三|四|五|六|日|天)?$`, text); matched {
		rule := &RecurrenceRule{Type: RecurrenceTypeWeekly, Interval: 1, Weekdays: []int{1, 2, 3, 4, 5}} // Default weekdays

		// Check for specific weekday
		weekdayMap := map[string]int{
			"一": 1, "二": 2, "三": 3, "四": 4, "五": 5,
			"六": 6, "日": 7, "天": 7,
		}
		if dayStr := regexp.MustCompile(`(周|星期)(一|二|三|四|五|六|日|天)`).FindStringSubmatch(text); len(dayStr) > 2 {
			day := dayStr[2]
			if weekdayNum, ok := weekdayMap[day]; ok {
				rule.Weekdays = []int{weekdayNum}
			}
		}

		// Check for interval
		if parts := regexp.MustCompile(`(\d+)`).FindStringSubmatch(text); len(parts) > 1 {
			if interval := parseInt(parts[1]); interval > 0 {
				rule.Interval = interval
			}
		}

		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("invalid recurrence rule: %w", err)
		}
		return rule, nil
	}

	// Monthly patterns
	if matched, _ := regexp.MatchString(`^每(月)(\d{1,2})号?$`, text); matched {
		rule := &RecurrenceRule{Type: RecurrenceTypeMonthly, MonthDay: 0, Interval: 1}
		if parts := regexp.MustCompile(`(\d{1,2})`).FindStringSubmatch(text); len(parts) > 1 {
			if day := parseInt(parts[1]); day >= 1 && day <= 31 {
				rule.MonthDay = day
			}
		}
		if rule.MonthDay == 0 {
			return nil, fmt.Errorf("invalid day of month: %s", text)
		}
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("invalid recurrence rule: %w", err)
		}
		return rule, nil
	}

	return nil, fmt.Errorf("unsupported recurrence pattern: %s", text)
}

// GenerateInstances generates all occurrence timestamps within a time range.
// startTs: The start timestamp of the first occurrence (Unix timestamp, always UTC)
// endTs: The end timestamp for generating instances (0 means no limit)
// Returns a slice of timestamps for each occurrence (Unix timestamps).
//
// Note on timezone handling:
// - Unix timestamps are always UTC
// - The startTs is assumed to be pre-converted to UTC by the caller based on user's timezone
// - For example, if user in Asia/Shanghai (UTC+8) schedules 9:00 AM, startTs is 01:00 UTC
// - Instances are generated using UTC arithmetic (adding 24h for daily, 7 days for weekly, etc.)
// - This ensures consistent behavior across timezones and simplifies the implementation
//
// For proper timezone-aware scheduling (e.g., "9:00 AM local time" regardless of DST changes),
// the caller should handle timezone conversion before/after calling this method.
func (r *RecurrenceRule) GenerateInstances(startTs int64, endTs int64) []int64 {
	var instances []int64

	// Safety check
	if startTs <= 0 {
		return instances
	}

	// Use UTC for consistent calculations
	startTime := time.Unix(startTs, 0).UTC()
	endTime := time.Now().UTC().Add(365 * 24 * time.Hour) // Default to 1 year limit

	if endTs > 0 {
		endTime = time.Unix(endTs, 0).UTC()
	}

	switch r.Type {
	case RecurrenceTypeDaily:
		instances = r.generateDailyInstances(startTime, endTime)

	case RecurrenceTypeWeekly:
		instances = r.generateWeeklyInstances(startTime, endTime)

	case RecurrenceTypeMonthly:
		instances = r.generateMonthlyInstances(startTime, endTime)
	}

	return instances
}

// generateDailyInstances generates instances for daily recurrence.
func (r *RecurrenceRule) generateDailyInstances(start, end time.Time) []int64 {
	var instances []int64
	current := start

	// Prevent infinite loops - limit to 10 years or 1000 instances
	maxInstances := 1000
	count := 0

	for current.Before(end) || current.Equal(end) {
		if count >= maxInstances {
			break
		}

		instances = append(instances, current.Unix())
		current = current.AddDate(0, 0, r.Interval)
		count++
	}

	return instances
}

// generateWeeklyInstances generates instances for weekly recurrence.
func (r *RecurrenceRule) generateWeeklyInstances(start, end time.Time) []int64 {
	var instances []int64
	current := start

	maxInstances := 520 // ~10 years
	count := 0

	for current.Before(end) || current.Equal(end) {
		if count >= maxInstances {
			break
		}

		// Check if current day matches target weekdays
		if r.matchesWeekday(current) {
			instances = append(instances, current.Unix())
		}

		// Move to next week
		current = current.AddDate(0, 0, 7*r.Interval)
		count++
	}

	return instances
}

// generateMonthlyInstances generates instances for monthly recurrence.
func (r *RecurrenceRule) generateMonthlyInstances(start, end time.Time) []int64 {
	var instances []int64
	// Iterate from 1st of start month to avoid skipping months when adding days to 31st
	// Use start.Location() to preserve timezone information
	current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())

	maxInstances := 120 // ~10 years
	count := 0

	for current.Before(end) || current.Equal(end) {
		if count >= maxInstances {
			break
		}

		// Find the target day in this month
		targetDay := r.MonthDay
		if targetDay > 28 {
			// Adjust for months with fewer days
			lastDay := getLastDayOfMonth(current.Year(), current.Month())
			if targetDay > lastDay {
				targetDay = lastDay
			}
		}

		// Create date for target day, preserving the original timezone
		instanceTime := time.Date(current.Year(), current.Month(), targetDay, 0, 0, 0, 0, start.Location())

		// Only add if it's the same or after start time
		if instanceTime.Equal(start) || instanceTime.After(start) {
			instances = append(instances, instanceTime.Unix())
		}

		// Move to next month
		current = current.AddDate(0, r.Interval, 0)
		count++
	}

	return instances
}

// matchesWeekday checks if the given time matches the target weekdays.
func (r *RecurrenceRule) matchesWeekday(t time.Time) bool {
	weekdays := r.Weekdays
	if len(weekdays) == 0 {
		// Default to all weekdays (Mon-Fri)
		weekdays = []int{1, 2, 3, 4, 5}
	}

	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Convert Sunday from 0 to 7
	}

	for _, target := range weekdays {
		if target == weekday {
			return true
		}
	}
	return false
}

// ToJSON converts the recurrence rule to JSON string.
func (r *RecurrenceRule) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ParseRecurrenceRuleFromJSON parses a recurrence rule from JSON string.
func ParseRecurrenceRuleFromJSON(jsonStr string) (*RecurrenceRule, error) {
	var rule RecurrenceRule
	if err := json.Unmarshal([]byte(jsonStr), &rule); err != nil {
		return nil, err
	}
	// Normalize type to lowercase to handle LLM variations (Daily, DAILY, etc.)
	rule.Type = RecurrenceType(strings.ToLower(string(rule.Type)))
	return &rule, nil
}

// ParseRecurrence is an alias for ParseRecurrenceRuleFromJSON.
func ParseRecurrence(jsonStr string) (*RecurrenceRule, error) {
	return ParseRecurrenceRuleFromJSON(jsonStr)
}

// getLastDayOfMonth returns the last day of the month.
func getLastDayOfMonth(year int, month time.Month) int {
	// First day of next month minus 1 day
	firstOfMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	return firstOfMonth.AddDate(0, 0, -1).Day()
}

// parseInt parses an integer from string (for recurrence interval/day).
func parseInt(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return 1
	}
	return val
}

// RecurrenceIterator provides lazy-loading iteration over recurrence instances.
// This is more memory-efficient than generating all instances upfront.
type RecurrenceIterator struct {
	rule       *RecurrenceRule
	cache      []int64
	startTs    int64
	cacheEndTs int64
	maxCache   int
	mu         sync.Mutex
	exhausted  bool
}

// Iterator creates a new iterator for this recurrence rule.
func (r *RecurrenceRule) Iterator(startTs int64) *RecurrenceIterator {
	return &RecurrenceIterator{
		rule:       r,
		startTs:    startTs,
		cache:      make([]int64, 0, 100),
		cacheEndTs: startTs - 1, // Start with empty cache
		maxCache:   500,
		exhausted:  false,
	}
}

// GetUntil returns all instances up to the given end timestamp.
// Results are cached for efficient repeated calls with increasing ranges.
func (it *RecurrenceIterator) GetUntil(endTs int64) []int64 {
	it.mu.Lock()
	defer it.mu.Unlock()

	// If already exhausted, return cached results
	if it.exhausted {
		return it.filterCached(endTs)
	}

	// If cache already covers the range, return filtered results
	if it.cacheEndTs >= endTs {
		return it.filterCached(endTs)
	}

	// Expand cache until we cover endTs or hit a limit
	for it.cacheEndTs < endTs && len(it.cache) < it.maxCache {
		next := it.generateNext()
		if next == 0 {
			it.exhausted = true
			break
		}
		it.cache = append(it.cache, next)
		it.cacheEndTs = next
	}

	return it.filterCached(endTs)
}

// Next returns the next instance in the sequence, or 0 if exhausted.
func (it *RecurrenceIterator) Next() int64 {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.exhausted && len(it.cache) > 0 {
		// Return cached value
		next := it.cache[0]
		it.cache = it.cache[1:]
		return next
	}

	if it.exhausted {
		return 0
	}

	next := it.generateNext()
	if next == 0 {
		it.exhausted = true
		return 0
	}

	return next
}

// CountInRange returns the number of instances in the given range without
// materializing all of them.
func (it *RecurrenceIterator) CountInRange(startTs, endTs int64) int {
	instances := it.GetUntil(endTs)
	count := 0
	for _, ts := range instances {
		if ts >= startTs && ts <= endTs {
			count++
		}
	}
	return count
}

// filterCached returns cached instances up to endTs.
func (it *RecurrenceIterator) filterCached(endTs int64) []int64 {
	result := make([]int64, 0)
	for _, ts := range it.cache {
		if ts <= endTs {
			result = append(result, ts)
		}
	}
	return result
}

// generateNext generates the next timestamp in the sequence.
// Returns 0 if exhausted.
func (it *RecurrenceIterator) generateNext() int64 {
	if it.exhausted {
		return 0
	}

	var next time.Time

	// Determine where to start from
	if len(it.cache) == 0 {
		// First instance
		next = time.Unix(it.startTs, 0).UTC()
	} else {
		// Subsequent instances
		lastTs := it.cache[len(it.cache)-1]
		lastTime := time.Unix(lastTs, 0).UTC()

		switch it.rule.Type {
		case RecurrenceTypeDaily:
			next = lastTime.AddDate(0, 0, it.rule.Interval)

		case RecurrenceTypeWeekly:
			// Find next matching weekday
			next = it.findNextWeekly(lastTime)

		case RecurrenceTypeMonthly:
			// Find next matching month day
			next = it.findNextMonthly(lastTime)

		default:
			return 0
		}
	}

	// Safety limit: don't generate beyond 10 years from start
	maxTime := time.Unix(it.startTs, 0).AddDate(10, 0, 0)
	if next.After(maxTime) {
		it.exhausted = true
		return 0
	}

	return next.Unix()
}

// findNextWeekly finds the next occurrence for weekly recurrence.
func (it *RecurrenceIterator) findNextWeekly(lastTime time.Time) time.Time {
	weekdays := it.rule.Weekdays
	if len(weekdays) == 0 {
		weekdays = []int{1, 2, 3, 4, 5} // Default to weekdays
	}

	// Move forward by interval weeks, then find next matching day
	candidate := lastTime.AddDate(0, 0, 7*it.rule.Interval)

	// Find the first matching weekday on or after candidate
	for attempts := 0; attempts < 7; attempts++ {
		weekday := int(candidate.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		for _, target := range weekdays {
			if weekday == target {
				return candidate
			}
		}
		candidate = candidate.AddDate(0, 0, 1)
	}

	return candidate
}

// findNextMonthly finds the next occurrence for monthly recurrence.
func (it *RecurrenceIterator) findNextMonthly(lastTime time.Time) time.Time {
	targetDay := it.rule.MonthDay

	// Move to next month
	nextMonth := lastTime.AddDate(0, it.rule.Interval, 0)

	// Get first day of that month
	firstOfMonth := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Get last day of the target month
	lastDay := getLastDayOfMonth(firstOfMonth.Year(), firstOfMonth.Month())

	// Adjust target day if it exceeds month length
	if targetDay > lastDay {
		targetDay = lastDay
	}

	// If we're past the target day in the original month, we need next month
	if lastTime.Day() > targetDay {
		firstOfMonth = firstOfMonth.AddDate(0, it.rule.Interval, 0)
		lastDay = getLastDayOfMonth(firstOfMonth.Year(), firstOfMonth.Month())
		if targetDay > lastDay {
			targetDay = lastDay
		}
	}

	return time.Date(firstOfMonth.Year(), firstOfMonth.Month(), targetDay,
		lastTime.Hour(), lastTime.Minute(), lastTime.Second(), 0, time.UTC)
}

// Reset resets the iterator to the beginning.
func (it *RecurrenceIterator) Reset() {
	it.mu.Lock()
	defer it.mu.Unlock()

	it.cache = make([]int64, 0, 100)
	it.cacheEndTs = it.startTs - 1
	it.exhausted = false
}
