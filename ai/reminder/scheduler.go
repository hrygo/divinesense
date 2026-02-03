// Package reminder provides reminder management for schedules and todos.
package reminder

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Scheduler runs background tasks for processing due reminders.
type Scheduler struct {
	service       *Service
	stopCh        chan struct{}
	logger        *slog.Logger
	processedChan chan int
	wg            sync.WaitGroup
	interval      time.Duration
	maxRetries    int
	retryDelay    time.Duration
	batchSize     int
	mu            sync.Mutex
	running       bool
}

// SchedulerConfig holds configuration for the scheduler.
type SchedulerConfig struct {
	Interval   time.Duration // How often to check for due reminders
	MaxRetries int           // Max retries for failed reminders
	RetryDelay time.Duration // Delay between retries
	BatchSize  int           // Max reminders to process per cycle
}

// DefaultSchedulerConfig returns default scheduler configuration.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		Interval:   time.Minute,
		MaxRetries: 3,
		RetryDelay: 5 * time.Second,
		BatchSize:  100,
	}
}

// NewScheduler creates a new reminder scheduler.
func NewScheduler(service *Service, config SchedulerConfig) *Scheduler {
	if config.Interval <= 0 {
		config.Interval = time.Minute
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = 3
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}

	return &Scheduler{
		service:    service,
		interval:   config.Interval,
		maxRetries: config.MaxRetries,
		retryDelay: config.RetryDelay,
		batchSize:  config.BatchSize,
		stopCh:     make(chan struct{}),
		logger:     slog.Default(),
	}
}

// Start begins the scheduler loop.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run(ctx)

	s.logger.Info("reminder scheduler started", "interval", s.interval)
	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	s.mu.Unlock()

	s.wg.Wait()
	s.logger.Info("reminder scheduler stopped")
}

// IsRunning returns whether the scheduler is running.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// SetLogger sets a custom logger.
func (s *Scheduler) SetLogger(logger *slog.Logger) {
	s.logger = logger
}

// EnableTestMode enables test mode with a channel for processed counts.
func (s *Scheduler) EnableTestMode() <-chan int {
	s.processedChan = make(chan int, 100)
	return s.processedChan
}

// run is the main scheduler loop.
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Process immediately on start
	s.processCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler context cancelled")
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processCycle(ctx)
		}
	}
}

// processCycle runs one cycle of reminder processing.
func (s *Scheduler) processCycle(ctx context.Context) {
	processed, err := s.service.ProcessDueReminders(ctx)
	if err != nil {
		s.logger.Error("failed to process due reminders", "error", err)
		return
	}

	if processed > 0 {
		s.logger.Info("processed due reminders", "count", processed)
	}

	// Report to test channel if enabled
	if s.processedChan != nil {
		select {
		case s.processedChan <- processed:
		default:
			// Don't block if channel is full
		}
	}
}

// RunOnce processes due reminders once (for manual triggering).
func (s *Scheduler) RunOnce(ctx context.Context) (int, error) {
	return s.service.ProcessDueReminders(ctx)
}

// Worker handles reminder processing with retry logic.
type Worker struct {
	store      ReminderStore
	notifier   Notifier
	logger     *slog.Logger
	maxRetries int
	retryDelay time.Duration
}

// NewWorker creates a new reminder worker.
func NewWorker(store ReminderStore, notifier Notifier, maxRetries int) *Worker {
	return &Worker{
		store:      store,
		notifier:   notifier,
		maxRetries: maxRetries,
		retryDelay: 5 * time.Second,
		logger:     slog.Default(),
	}
}

// ProcessReminder processes a single reminder with retry logic.
func (w *Worker) ProcessReminder(ctx context.Context, reminder *Reminder) error {
	var lastErr error

	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			w.logger.Info("retrying reminder",
				"reminder_id", reminder.ID,
				"attempt", attempt,
				"max_retries", w.maxRetries,
			)
			time.Sleep(w.retryDelay)
		}

		// Try to send through all channels
		allSuccess := true
		for _, channel := range reminder.Channels {
			if err := w.notifier.Send(ctx, reminder.UserID, channel, reminder.Message); err != nil {
				lastErr = err
				allSuccess = false
				w.logger.Warn("failed to send via channel",
					"reminder_id", reminder.ID,
					"channel", channel,
					"error", err,
				)
			}
		}

		if allSuccess {
			return w.store.MarkSent(ctx, reminder.ID)
		}
	}

	// All retries failed
	if lastErr != nil {
		_ = w.store.MarkFailed(ctx, reminder.ID, lastErr.Error())
	}

	return lastErr
}

// ProcessBatch processes a batch of reminders.
func (w *Worker) ProcessBatch(ctx context.Context, reminders []*Reminder) (processed int, failed int) {
	for _, r := range reminders {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := w.ProcessReminder(ctx, r); err != nil {
			failed++
		} else {
			processed++
		}
	}
	return
}

// HealthCheck provides health check for the scheduler.
type HealthCheck struct {
	scheduler  *Scheduler
	lastCheck  time.Time
	checkCount int64
	mu         sync.RWMutex
}

// NewHealthCheck creates a new health check for the scheduler.
func NewHealthCheck(scheduler *Scheduler) *HealthCheck {
	return &HealthCheck{
		scheduler: scheduler,
	}
}

// Check returns the health status.
func (h *HealthCheck) Check() HealthStatus {
	h.mu.Lock()
	h.lastCheck = time.Now()
	h.checkCount++
	h.mu.Unlock()

	return HealthStatus{
		Healthy:    h.scheduler.IsRunning(),
		LastCheck:  h.lastCheck,
		CheckCount: h.checkCount,
	}
}

// HealthStatus represents the health of the scheduler.
type HealthStatus struct {
	LastCheck  time.Time `json:"last_check"`
	CheckCount int64     `json:"check_count"`
	Healthy    bool      `json:"healthy"`
}

// Stats holds scheduler statistics.
type Stats struct {
	LastRunAt      time.Time `json:"last_run_at"`
	TotalProcessed int64     `json:"total_processed"`
	TotalFailed    int64     `json:"total_failed"`
	AverageLatency float64   `json:"average_latency_ms"`
}

// MetricsCollector collects scheduler metrics.
type MetricsCollector struct {
	stats Stats
	mu    sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// RecordProcessed records processed reminders.
func (m *MetricsCollector) RecordProcessed(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.TotalProcessed += int64(count)
	m.stats.LastRunAt = time.Now()
}

// RecordFailed records failed reminders.
func (m *MetricsCollector) RecordFailed(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.TotalFailed += int64(count)
}

// GetStats returns current statistics.
func (m *MetricsCollector) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}
