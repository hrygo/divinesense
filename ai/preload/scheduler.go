// Package preload provides predictive cache preloading scheduling.
// This file implements the scheduler that performs preloading based on predictions.
package preload

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Scheduler manages predictive preloading tasks.
type Scheduler struct {
	analyzer  *Analyzer
	preloader Preloader
	mu        sync.RWMutex
	tasks     map[int64]*scheduledTask
	ticker    *time.Ticker
	stopCh    chan struct{}
	running   atomic.Bool
	config    SchedulerConfig
	stats     *schedulerStats
}

// Preloader performs the actual preloading of data.
type Preloader interface {
	// PreloadQuery preloads data for a query.
	PreloadQuery(ctx context.Context, userID int64, query string) error

	// PreloadUserContext preloads user context data.
	PreloadUserContext(ctx context.Context, userID int64) error
}

// scheduledTask represents a scheduled preload task.
type scheduledTask struct {
	UserID      int64
	Suggestions *PreloadSuggestions
	LastRun     time.Time
	NextRun     time.Time
	RunCount    int
}

// SchedulerConfig configures the preload scheduler.
type SchedulerConfig struct {
	// AnalysisInterval is how often to analyze patterns.
	AnalysisInterval time.Duration

	// PreloadInterval is how often to perform preloading.
	PreloadInterval time.Duration

	// QuietHours are hours when preloading is paused.
	QuietHours []int // 0-23

	// MaxCPUPercent limits CPU usage for preloading.
	MaxCPUPercent int

	// MaxMemoryMB limits memory usage for preloading.
	MaxMemoryMB int

	// ConcurrentPreloads is the max concurrent preload operations.
	ConcurrentPreloads int
}

// DefaultSchedulerConfig returns default scheduler configuration.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		AnalysisInterval:   24 * time.Hour,          // Daily analysis
		PreloadInterval:    1 * time.Hour,           // Hourly preloading
		QuietHours:         []int{0, 1, 2, 3, 4, 5}, // 12am-5am
		MaxCPUPercent:      10,                      // Max 10% CPU
		MaxMemoryMB:        100,                     // Max 100MB
		ConcurrentPreloads: 3,
	}
}

// NewScheduler creates a new preload scheduler.
func NewScheduler(analyzer *Analyzer, preloader Preloader, cfg SchedulerConfig) *Scheduler {
	if cfg.AnalysisInterval <= 0 {
		cfg.AnalysisInterval = DefaultSchedulerConfig().AnalysisInterval
	}
	if cfg.PreloadInterval <= 0 {
		cfg.PreloadInterval = DefaultSchedulerConfig().PreloadInterval
	}
	if cfg.MaxCPUPercent <= 0 {
		cfg.MaxCPUPercent = DefaultSchedulerConfig().MaxCPUPercent
	}
	if cfg.MaxMemoryMB <= 0 {
		cfg.MaxMemoryMB = DefaultSchedulerConfig().MaxMemoryMB
	}
	if cfg.ConcurrentPreloads <= 0 {
		cfg.ConcurrentPreloads = DefaultSchedulerConfig().ConcurrentPreloads
	}

	return &Scheduler{
		analyzer:  analyzer,
		preloader: preloader,
		tasks:     make(map[int64]*scheduledTask),
		stopCh:    make(chan struct{}),
		config:    cfg,
		stats:     &schedulerStats{},
	}
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	if !s.running.CompareAndSwap(false, true) {
		return // Already running
	}

	s.ticker = time.NewTicker(s.config.PreloadInterval)

	go s.run()
	slog.Info("preload scheduler started",
		"interval", s.config.PreloadInterval,
		"quiet_hours", s.config.QuietHours,
	)
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	if !s.running.CompareAndSwap(true, false) {
		return // Not running
	}

	close(s.stopCh)
	if s.ticker != nil {
		s.ticker.Stop()
	}

	slog.Info("preload scheduler stopped")
}

// run is the main scheduler loop.
func (s *Scheduler) run() {
	for {
		select {
		case <-s.ticker.C:
			s.tick()
		case <-s.stopCh:
			return
		}
	}
}

// tick performs a single scheduler tick.
func (s *Scheduler) tick() {
	ctx := context.Background()

	// Check if we're in quiet hours
	if s.isQuietHour() {
		slog.Debug("skipping preload during quiet hours")
		return
	}

	// Check resource limits
	if !s.checkResources() {
		slog.Warn("skipping preload due to resource limits")
		return
	}

	// Get all cached patterns and schedule preloads
	s.mu.Lock()
	defer s.mu.Unlock()

	// Collect users that need preloading
	var toPreload []*scheduledTask
	for _, task := range s.tasks {
		if time.Now().After(task.NextRun) {
			toPreload = append(toPreload, task)
		}
	}

	// Run preloads with concurrency limit
	sem := make(chan struct{}, s.config.ConcurrentPreloads)
	var wg sync.WaitGroup

	for _, task := range toPreload {
		wg.Add(1)
		go func(t *scheduledTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := s.preloadForUser(ctx, t); err != nil {
				slog.Warn("preload failed",
					"user_id", t.UserID,
					"error", err,
				)
				s.recordError()
			} else {
				s.recordSuccess()
			}

			t.LastRun = time.Now()
			t.NextRun = time.Now().Add(s.config.PreloadInterval)
			t.RunCount++
		}(task)
	}

	wg.Wait()
}

// preloadForUser performs preloading for a single user.
func (s *Scheduler) preloadForUser(ctx context.Context, task *scheduledTask) error {
	// Get fresh suggestions
	suggestions := s.analyzer.GetPreloadSuggestions(ctx, task.UserID)

	task.Suggestions = suggestions

	// Preload user context
	if err := s.preloader.PreloadUserContext(ctx, task.UserID); err != nil {
		slog.Warn("failed to preload user context", "user_id", task.UserID, "error", err)
	}

	// Preload top queries
	maxQueries := 3
	for i, query := range suggestions.Queries {
		if i >= maxQueries {
			break
		}
		if err := s.preloader.PreloadQuery(ctx, task.UserID, query); err != nil {
			slog.Debug("failed to preload query",
				"user_id", task.UserID,
				"query", query,
				"error", err,
			)
		}
	}

	return nil
}

// ScheduleUser adds a user to the preload schedule.
func (s *Scheduler) ScheduleUser(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := &scheduledTask{
		UserID:  userID,
		NextRun: time.Now(), // Run immediately on next tick
	}

	s.tasks[userID] = task
	slog.Debug("user scheduled for preload", "user_id", userID)
}

// UnscheduleUser removes a user from the preload schedule.
func (s *Scheduler) UnscheduleUser(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tasks, userID)
	slog.Debug("user unscheduled from preload", "user_id", userID)
}

// isQuietHour checks if current time is in quiet hours.
func (s *Scheduler) isQuietHour() bool {
	hour := time.Now().Hour()
	for _, qh := range s.config.QuietHours {
		if hour == qh {
			return true
		}
	}
	return false
}

// checkResources checks if system resources are available for preloading.
func (s *Scheduler) checkResources() bool {
	// Check CPU usage
	if s.config.MaxCPUPercent > 0 {
		cpuPercent := getCPUPercent()
		if cpuPercent > s.config.MaxCPUPercent {
			return false
		}
	}

	// Check memory usage
	if s.config.MaxMemoryMB > 0 {
		memMB := getMemoryUsageMB()
		if memMB > s.config.MaxMemoryMB {
			return false
		}
	}

	return true
}

// recordSuccess records a successful preload.
func (s *Scheduler) recordSuccess() {
	atomic.AddInt64(&s.stats.successCount, 1)
}

// recordError records a preload error.
func (s *Scheduler) recordError() {
	atomic.AddInt64(&s.stats.errorCount, 1)
}

// GetStats returns scheduler statistics.
func (s *Scheduler) GetStats() *SchedulerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &SchedulerStats{
		ScheduledUsers: len(s.tasks),
		SuccessCount:   atomic.LoadInt64(&s.stats.successCount),
		ErrorCount:     atomic.LoadInt64(&s.stats.errorCount),
		Running:        s.running.Load(),
		QuietHours:     s.config.QuietHours,
		MaxCPUPercent:  s.config.MaxCPUPercent,
		MaxMemoryMB:    s.config.MaxMemoryMB,
	}
}

// SchedulerStats contains scheduler statistics.
type SchedulerStats struct {
	ScheduledUsers int
	SuccessCount   int64
	ErrorCount     int64
	Running        bool
	QuietHours     []int
	MaxCPUPercent  int
	MaxMemoryMB    int
}

type schedulerStats struct {
	successCount int64
	errorCount   int64
}

// getCPUPercent returns current CPU usage percentage (simplified).
func getCPUPercent() int {
	// Simplified - in production use proper CPU monitoring
	var usage runtime.MemStats
	runtime.ReadMemStats(&usage)
	// This is a placeholder - real implementation would use system CPU stats
	return 5
}

// getMemoryUsageMB returns current memory usage in MB.
func getMemoryUsageMB() int {
	var usage runtime.MemStats
	runtime.ReadMemStats(&usage)
	return int(usage.Alloc / 1024 / 1024)
}

// TriggerPreload manually triggers preloading for a user.
func (s *Scheduler) TriggerPreload(ctx context.Context, userID int64) error {
	s.mu.Lock()
	task, exists := s.tasks[userID]
	if !exists {
		task = &scheduledTask{
			UserID:  userID,
			NextRun: time.Now(),
		}
		s.tasks[userID] = task
	}
	s.mu.Unlock()

	return s.preloadForUser(ctx, task)
}

// UpdateConfig updates the scheduler configuration.
func (s *Scheduler) UpdateConfig(cfg SchedulerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = cfg

	// Update ticker if interval changed
	if s.ticker != nil {
		s.ticker.Reset(s.config.PreloadInterval)
	}
}

// GetConfig returns the current scheduler configuration.
func (s *Scheduler) GetConfig() SchedulerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetScheduledUsers returns the list of scheduled user IDs.
func (s *Scheduler) GetScheduledUsers() []int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]int64, 0, len(s.tasks))
	for userID := range s.tasks {
		users = append(users, userID)
	}
	return users
}

// IsRunning returns true if the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	return s.running.Load()
}

// MockPreloader is a mock preloader for testing.
type MockPreloader struct {
	Calls []struct {
		UserID int64
		Query  string
	}
	mu sync.Mutex
}

// NewMockPreloader creates a new mock preloader.
func NewMockPreloader() *MockPreloader {
	return &MockPreloader{
		Calls: make([]struct {
			UserID int64
			Query  string
		}, 0),
	}
}

// PreloadQuery records the preload call.
func (m *MockPreloader) PreloadQuery(ctx context.Context, userID int64, query string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, struct {
		UserID int64
		Query  string
	}{userID, query})
	return nil
}

// PreloadUserContext records the preload call.
func (m *MockPreloader) PreloadUserContext(ctx context.Context, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = append(m.Calls, struct {
		UserID int64
		Query  string
	}{userID, ""})
	return nil
}

// GetCallCount returns the number of preload calls.
func (m *MockPreloader) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Calls)
}

// Clear clears the call history.
func (m *MockPreloader) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = make([]struct {
		UserID int64
		Query  string
	}, 0)
}
