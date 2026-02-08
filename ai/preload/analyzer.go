// Package preload provides predictive cache preloading for AI operations.
// It implements Issue #102: Predictive preloading based on user patterns.
package preload

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"
)

// Analyzer analyzes user behavior patterns for predictive preloading.
type Analyzer struct {
	store        PatternStore
	mu           sync.RWMutex
	cache        map[int64]*UserPattern // userID -> pattern
	maxCacheSize int
	ttl          time.Duration
}

// PatternStore persists and retrieves user patterns.
type PatternStore interface {
	// SavePattern saves a user pattern.
	SavePattern(ctx context.Context, userID int64, pattern *UserPattern) error

	// LoadPattern loads a user pattern.
	LoadPattern(ctx context.Context, userID int64) (*UserPattern, error)

	// UpdatePattern updates an existing pattern.
	UpdatePattern(ctx context.Context, userID int64, pattern *UserPattern) error
}

// UserPattern represents a user's behavioral pattern.
type UserPattern struct {
	// UserID is the user's ID.
	UserID int64

	// ActiveHours shows when the user is most active.
	// Each bucket represents an hour of the day (0-23).
	ActiveHours [24]int

	// ActiveDays shows which days of the week the user is active.
	// Each bucket represents a day (0=Sunday, 6=Saturday).
	ActiveDays [7]int

	// FrequentQueries are the user's most common search queries.
	FrequentQueries []*QueryPattern

	// CommonTopics are topics the user frequently searches for.
	CommonTopics []string

	// LastUpdate is when this pattern was last updated.
	LastUpdate time.Time

	// SampleCount is the number of data points used to build this pattern.
	SampleCount int64

	// AvgSessionDuration is the user's average session duration.
	AvgSessionDuration time.Duration

	// PeakHours are the user's peak activity hours.
	PeakHours []int

	// QuietHours are hours when the user is rarely active.
	QuietHours []int
}

// QueryPattern represents a frequent search query pattern.
type QueryPattern struct {
	// Query is the search query (may be normalized).
	Query string

	// Frequency is how often this query is used.
	Frequency int

	// LastUsed is when this query was last used.
	LastUsed time.Time

	// AvgTokens is the average tokens used for this query.
	AvgTokens int
}

// Config configures the analyzer.
type Config struct {
	// Store is the pattern store backend.
	Store PatternStore

	// MaxCacheSize is the maximum patterns to keep in memory.
	MaxCacheSize int

	// TTL is how long cached patterns remain valid.
	TTL time.Duration

	// MinSamples is the minIntimum samples before a pattern is considered valid.
	MinSamples int
}

// DefaultConfig returns default analyzer configuration.
func DefaultConfig() Config {
	return Config{
		MaxCacheSize: 1000,
		TTL:          24 * time.Hour,
		MinSamples:   10,
	}
}

// NewAnalyzer creates a new pattern analyzer.
func NewAnalyzer(cfg Config) *Analyzer {
	if cfg.MaxCacheSize <= 0 {
		cfg.MaxCacheSize = DefaultConfig().MaxCacheSize
	}
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultConfig().TTL
	}
	if cfg.MinSamples <= 0 {
		cfg.MinSamples = DefaultConfig().MinSamples
	}

	return &Analyzer{
		store:        cfg.Store,
		cache:        make(map[int64]*UserPattern),
		maxCacheSize: cfg.MaxCacheSize,
		ttl:          cfg.TTL,
	}
}

// RecordActivity records a user activity for pattern analysis.
func (a *Analyzer) RecordActivity(ctx context.Context, userID int64, activity *Activity) error {
	pattern, err := a.getOrCreatePattern(ctx, userID)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Update active hours
	hour := activity.Timestamp.Hour()
	pattern.ActiveHours[hour]++
	pattern.SampleCount++

	// Update active days
	day := activity.Timestamp.Weekday()
	pattern.ActiveDays[day]++

	// Update queries
	if activity.Query != "" {
		a.updateQueryPattern(pattern, activity)
	}

	// Update session duration
	if activity.SessionDuration > 0 {
		pattern.AvgSessionDuration = a.updateAverage(
			pattern.AvgSessionDuration,
			pattern.SampleCount,
			int64(activity.SessionDuration),
		)
	}

	// Recompute peak and quiet hours
	a.recomputeHours(pattern)

	pattern.LastUpdate = time.Now()

	// Save asynchronously
	go func() {
		if a.store != nil {
			ctx := context.Background()
			if err := a.store.SavePattern(ctx, userID, pattern); err != nil {
				slog.Error("failed to save user pattern", "user_id", userID, "error", err)
			}
		}
	}()

	return nil
}

// Activity represents a single user activity.
type Activity struct {
	// Timestamp when the activity occurred.
	Timestamp time.Time

	// Query is the search query (if any).
	Query string

	// AgentType used for the query.
	AgentType string

	// SessionDuration is the duration of the session.
	SessionDuration time.Duration

	// TokensUsed is the number of tokens consumed.
	TokensUsed int

	// Topics are topics related to this activity.
	Topics []string
}

// updateQueryPattern updates the query patterns based on activity.
func (a *Analyzer) updateQueryPattern(pattern *UserPattern, activity *Activity) {
	normalized := normalizeQuery(activity.Query)

	// Find existing pattern
	for _, qp := range pattern.FrequentQueries {
		if qp.Query == normalized {
			qp.Frequency++
			qp.LastUsed = activity.Timestamp
			if activity.TokensUsed > 0 {
				qp.AvgTokens = int(a.updateAverage(
					time.Duration(qp.AvgTokens),
					int64(qp.Frequency-1),
					int64(activity.TokensUsed),
				))
			}
			return
		}
	}

	// Add new pattern
	pattern.FrequentQueries = append(pattern.FrequentQueries, &QueryPattern{
		Query:     normalized,
		Frequency: 1,
		LastUsed:  activity.Timestamp,
		AvgTokens: activity.TokensUsed,
	})

	// Keep only top queries
	if len(pattern.FrequentQueries) > 50 {
		sort.Slice(pattern.FrequentQueries, func(i, j int) bool {
			return pattern.FrequentQueries[i].Frequency > pattern.FrequentQueries[j].Frequency
		})
		pattern.FrequentQueries = pattern.FrequentQueries[:50]
	}
}

// recomputeHours recomputes peak and quiet hours.
func (a *Analyzer) recomputeHours(pattern *UserPattern) {
	// Find peak hours (top 3 by activity)
	type hourCount struct {
		hour  int
		count int
	}

	var hours []hourCount
	for h, count := range pattern.ActiveHours {
		if count > 0 {
			hours = append(hours, hourCount{hour: h, count: count})
		}
	}

	sort.Slice(hours, func(i, j int) bool {
		return hours[i].count > hours[j].count
	})

	// Top 3 hours as peak hours
	pattern.PeakHours = nil
	for i := 0; i < min(3, len(hours)); i++ {
		pattern.PeakHours = append(pattern.PeakHours, hours[i].hour)
	}

	// Hours with below-average activity as quiet hours
	if len(hours) > 0 {
		avg := 0
		for _, h := range hours {
			avg += h.count
		}
		avg /= len(hours)

		pattern.QuietHours = nil
		for _, h := range hours {
			if h.count < avg/2 {
				pattern.QuietHours = append(pattern.QuietHours, h.hour)
			}
		}
	}
}

// GetPattern returns the user's current pattern.
func (a *Analyzer) GetPattern(ctx context.Context, userID int64) (*UserPattern, error) {
	return a.getOrCreatePattern(ctx, userID)
}

// PredictNextQuery predicts the user's next likely queries.
func (a *Analyzer) PredictNextQuery(ctx context.Context, userID int64, currentQuery string) []string {
	pattern, err := a.GetPattern(ctx, userID)
	if err != nil {
		return nil
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	var predictions []string

	// Return top frequent queries
	for _, qp := range pattern.FrequentQueries {
		if qp.Query != currentQuery {
			predictions = append(predictions, qp.Query)
		}
		if len(predictions) >= 5 {
			break
		}
	}

	return predictions
}

// IsPeakHour returns true if current time is a peak hour for the user.
func (a *Analyzer) IsPeakHour(ctx context.Context, userID int64) bool {
	pattern, err := a.GetPattern(ctx, userID)
	if err != nil {
		return false
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	hour := time.Now().Hour()
	for _, peak := range pattern.PeakHours {
		if peak == hour {
			return true
		}
	}

	return false
}

// IsQuietHour returns true if current time is a quiet hour for the user.
func (a *Analyzer) IsQuietHour(ctx context.Context, userID int64) bool {
	pattern, err := a.GetPattern(ctx, userID)
	if err != nil {
		return false
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	hour := time.Now().Hour()
	for _, quiet := range pattern.QuietHours {
		if quiet == hour {
			return true
		}
	}

	return false
}

// GetPreloadSuggestions returns suggested items to preload for a user.
func (a *Analyzer) GetPreloadSuggestions(ctx context.Context, userID int64) *PreloadSuggestions {
	pattern, err := a.GetPattern(ctx, userID)
	if err != nil {
		return &PreloadSuggestions{}
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	suggestions := &PreloadSuggestions{
		UserID:    userID,
		Timestamp: time.Now(),
	}

	// Add top queries
	for i, qp := range pattern.FrequentQueries {
		if i >= 10 {
			break
		}
		suggestions.Queries = append(suggestions.Queries, qp.Query)
	}

	// Mark if currently peak hour
	suggestions.IsPeakHour = a.IsPeakHour(ctx, userID)

	// Expected tokens based on pattern
	if len(pattern.FrequentQueries) > 0 {
		totalTokens := 0
		count := 0
		for _, qp := range pattern.FrequentQueries {
			if qp.AvgTokens > 0 {
				totalTokens += qp.AvgTokens
				count++
			}
		}
		if count > 0 {
			suggestions.ExpectedTokens = totalTokens / count
		}
	}

	return suggestions
}

// PreloadSuggestions contains items suggested for preloading.
type PreloadSuggestions struct {
	UserID         int64
	Timestamp      time.Time
	Queries        []string
	IsPeakHour     bool
	ExpectedTokens int
}

// getOrCreatePattern gets or creates a user pattern.
func (a *Analyzer) getOrCreatePattern(ctx context.Context, userID int64) (*UserPattern, error) {
	a.mu.RLock()
	pattern, exists := a.cache[userID]
	a.mu.RUnlock()

	if exists && time.Since(pattern.LastUpdate) < a.ttl {
		return pattern, nil
	}

	// Load from store
	if a.store != nil {
		loaded, err := a.store.LoadPattern(ctx, userID)
		if err == nil && loaded != nil {
			a.mu.Lock()
			// Evict if cache is full
			if len(a.cache) >= a.maxCacheSize {
				a.evictOldest()
			}
			a.cache[userID] = loaded
			a.mu.Unlock()
			return loaded, nil
		}
	}

	// Create new pattern
	a.mu.Lock()
	defer a.mu.Unlock()

	if pattern, exists := a.cache[userID]; exists {
		return pattern, nil
	}

	newPattern := &UserPattern{
		UserID:          userID,
		LastUpdate:      time.Now(),
		FrequentQueries: make([]*QueryPattern, 0),
	}

	if len(a.cache) >= a.maxCacheSize {
		a.evictOldest()
	}

	a.cache[userID] = newPattern
	return newPattern, nil
}

// evictOldest removes the oldest pattern from cache.
func (a *Analyzer) evictOldest() {
	var oldestID int64
	var oldestTime time.Time

	for id, pattern := range a.cache {
		if oldestID == 0 || pattern.LastUpdate.Before(oldestTime) {
			oldestID = id
			oldestTime = pattern.LastUpdate
		}
	}

	if oldestID != 0 {
		delete(a.cache, oldestID)
	}
}

// updateAverage updates a running average.
func (a *Analyzer) updateAverage(current time.Duration, count, newValue int64) time.Duration {
	if count == 0 {
		return time.Duration(newValue)
	}

	oldAvg := int64(current)
	newAvg := (oldAvg*count + newValue) / (count + 1)
	return time.Duration(newAvg)
}

// normalizeQuery normalizes a query for pattern matching.
func normalizeQuery(query string) string {
	// Simple normalization: lowercase and trim
	// In production, you might do more sophisticated normalization
	return query
}

// Clear clears the pattern cache.
func (a *Analyzer) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache = make(map[int64]*UserPattern)
}

// GetStats returns analyzer statistics.
func (a *Analyzer) GetStats() *AnalyzerStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	stats := &AnalyzerStats{
		CachedPatterns: len(a.cache),
		MaxCacheSize:   a.maxCacheSize,
	}

	var totalSamples int64
	var totalQueries int

	for _, pattern := range a.cache {
		totalSamples += pattern.SampleCount
		totalQueries += len(pattern.FrequentQueries)
	}

	stats.TotalSamples = totalSamples
	stats.TotalQueries = totalQueries

	return stats
}

// AnalyzerStats contains analyzer statistics.
type AnalyzerStats struct {
	CachedPatterns int
	MaxCacheSize   int
	TotalSamples   int64
	TotalQueries   int
}

// ExportJSON exports the pattern as JSON for debugging.
func (p *UserPattern) ExportJSON() (string, error) {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CalculateEntropy calculates the entropy of the user's activity distribution.
// Higher entropy means more predictable patterns.
func (p *UserPattern) CalculateEntropy() float64 {
	total := 0
	for _, count := range p.ActiveHours {
		total += count
	}

	if total == 0 {
		return 0
	}

	entropy := 0.0
	for _, count := range p.ActiveHours {
		if count > 0 {
			prob := float64(count) / float64(total)
			entropy -= prob * math.Log2(prob)
		}
	}

	return entropy
}

// GetConfidence returns a confidence score (0-1) for the pattern.
// Higher values mean more reliable predictions.
func (p *UserPattern) GetConfidence(minIntSamples int) float64 {
	if p.SampleCount < int64(minIntSamples) {
		return float64(p.SampleCount) / float64(minIntSamples)
	}

	// Cap at 0.95 (never 100% confident)
	return 0.95
}

// Merge merges another pattern into this one.
func (p *UserPattern) Merge(other *UserPattern) {
	for h := 0; h < 24; h++ {
		p.ActiveHours[h] += other.ActiveHours[h]
	}

	for d := 0; d < 7; d++ {
		p.ActiveDays[d] += other.ActiveDays[d]
	}

	p.SampleCount += other.SampleCount

	// Merge query patterns
	for _, otherQP := range other.FrequentQueries {
		found := false
		for _, qp := range p.FrequentQueries {
			if qp.Query == otherQP.Query {
				qp.Frequency += otherQP.Frequency
				if otherQP.LastUsed.After(qp.LastUsed) {
					qp.LastUsed = otherQP.LastUsed
				}
				found = true
				break
			}
		}
		if !found {
			p.FrequentQueries = append(p.FrequentQueries, otherQP)
		}
	}

	p.LastUpdate = time.Now()
}

// MemoryPatternStore is an in-memory pattern store for testing.
type MemoryPatternStore struct {
	mu       sync.RWMutex
	patterns map[int64]*UserPattern
}

// NewMemoryPatternStore creates a new memory pattern store.
func NewMemoryPatternStore() *MemoryPatternStore {
	return &MemoryPatternStore{
		patterns: make(map[int64]*UserPattern),
	}
}

// SavePattern saves a pattern to memory.
func (m *MemoryPatternStore) SavePattern(ctx context.Context, userID int64, pattern *UserPattern) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.patterns[userID] = pattern
	return nil
}

// LoadPattern loads a pattern from memory.
func (m *MemoryPatternStore) LoadPattern(ctx context.Context, userID int64) (*UserPattern, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.patterns[userID], nil
}

// UpdatePattern updates a pattern in memory.
func (m *MemoryPatternStore) UpdatePattern(ctx context.Context, userID int64, pattern *UserPattern) error {
	return m.SavePattern(ctx, userID, pattern)
}
