package session

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	// DefaultRetentionDays is the default number of days to retain sessions.
	DefaultRetentionDays = 30
	// DefaultCleanupInterval is the default interval between cleanup runs.
	DefaultCleanupInterval = 24 * time.Hour
)

// CleanupConfig holds configuration for the cleanup job.
type CleanupConfig struct {
	RetentionDays   int           // Number of days to retain sessions (default: 30)
	CleanupInterval time.Duration // Interval between cleanup runs (default: 24h)
}

// DefaultCleanupConfig returns the default cleanup configuration.
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		RetentionDays:   DefaultRetentionDays,
		CleanupInterval: DefaultCleanupInterval,
	}
}

// SessionCleanupJob handles periodic cleanup of expired sessions.
type SessionCleanupJob struct {
	sessionSvc SessionService
	stopChan   chan struct{}
	config     CleanupConfig
	mu         sync.Mutex
	running    bool
}

// NewSessionCleanupJob creates a new cleanup job.
func NewSessionCleanupJob(svc SessionService, config CleanupConfig) *SessionCleanupJob {
	if config.RetentionDays <= 0 {
		config.RetentionDays = DefaultRetentionDays
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = DefaultCleanupInterval
	}

	return &SessionCleanupJob{
		sessionSvc: svc,
		config:     config,
	}
}

// Start begins the periodic cleanup job.
// This method is non-blocking and starts the cleanup in a goroutine.
func (j *SessionCleanupJob) Start(ctx context.Context) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.running {
		return nil // Already running
	}

	j.running = true
	j.stopChan = make(chan struct{})

	go j.run(ctx)

	slog.Info("session cleanup job started",
		"retention_days", j.config.RetentionDays,
		"interval", j.config.CleanupInterval)

	return nil
}

// Stop stops the cleanup job.
func (j *SessionCleanupJob) Stop() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.running {
		return
	}

	close(j.stopChan)
	j.running = false

	slog.Info("session cleanup job stopped")
}

// RunOnce executes a single cleanup run immediately.
// Useful for testing or manual cleanup.
func (j *SessionCleanupJob) RunOnce(ctx context.Context) (int64, error) {
	return j.cleanup(ctx)
}

// run is the main loop for the cleanup job.
func (j *SessionCleanupJob) run(ctx context.Context) {
	ticker := time.NewTicker(j.config.CleanupInterval)
	defer ticker.Stop()
	defer func() {
		// Ensure running state is cleared when goroutine exits
		j.mu.Lock()
		j.running = false
		j.mu.Unlock()
	}()

	// Run immediately on start
	if deleted, err := j.cleanup(ctx); err != nil {
		slog.Error("initial session cleanup failed", "error", err)
	} else if deleted > 0 {
		slog.Info("initial session cleanup completed", "deleted", deleted)
	}

	for {
		select {
		case <-ctx.Done():
			// Context cancelled - goroutine exiting, running state cleared by defer
			return
		case <-j.stopChan:
			// Explicitly stopped
			return
		case <-ticker.C:
			if deleted, err := j.cleanup(ctx); err != nil {
				slog.Error("session cleanup failed", "error", err)
			} else if deleted > 0 {
				slog.Info("session cleanup completed", "deleted", deleted)
			}
		}
	}
}

// cleanup performs the actual cleanup.
func (j *SessionCleanupJob) cleanup(ctx context.Context) (int64, error) {
	return j.sessionSvc.CleanupExpired(ctx, j.config.RetentionDays)
}

// IsRunning returns whether the cleanup job is currently running.
func (j *SessionCleanupJob) IsRunning() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.running
}
