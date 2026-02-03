package metrics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/hrygo/divinesense/store"
)

// Persister handles periodic persistence of aggregated metrics to the database.
type Persister struct {
	store      *store.Store
	aggregator *Aggregator

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	flushInterval   time.Duration
	retentionPeriod time.Duration
	cleanupInterval time.Duration
}

// PersisterConfig configures the metrics persister.
type PersisterConfig struct {
	FlushInterval   time.Duration // How often to flush metrics to DB (default: 1 hour)
	RetentionPeriod time.Duration // How long to keep metrics (default: 30 days)
	CleanupInterval time.Duration // How often to run cleanup (default: 24 hours)
}

// DefaultPersisterConfig returns default persister configuration.
func DefaultPersisterConfig() PersisterConfig {
	return PersisterConfig{
		FlushInterval:   time.Hour,
		RetentionPeriod: 30 * 24 * time.Hour,
		CleanupInterval: 24 * time.Hour,
	}
}

// NewPersister creates a new metrics persister.
func NewPersister(s *store.Store, agg *Aggregator, cfg PersisterConfig) *Persister {
	if cfg.FlushInterval == 0 {
		cfg.FlushInterval = time.Hour
	}
	if cfg.RetentionPeriod == 0 {
		cfg.RetentionPeriod = 30 * 24 * time.Hour
	}
	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = 24 * time.Hour
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Persister{
		store:           s,
		aggregator:      agg,
		ctx:             ctx,
		cancel:          cancel,
		flushInterval:   cfg.FlushInterval,
		retentionPeriod: cfg.RetentionPeriod,
		cleanupInterval: cfg.CleanupInterval,
	}
}

// Start begins the background persistence and cleanup tasks.
func (p *Persister) Start() {
	p.wg.Add(2)
	go p.flushLoop()
	go p.cleanupLoop()
}

// Close stops the persister and waits for goroutines to finish.
func (p *Persister) Close() {
	p.cancel()
	p.wg.Wait()
}

// Flush immediately persists all completed hour buckets to the database.
func (p *Persister) Flush(ctx context.Context) error {
	currentHour := truncateToHour(time.Now())

	// Flush agent metrics
	agentSnapshots := p.aggregator.FlushAgentMetrics(currentHour)
	for _, snapshot := range agentSnapshots {
		_, err := p.store.UpsertAgentMetrics(ctx, &store.UpsertAgentMetrics{
			HourBucket:   snapshot.HourBucket,
			AgentType:    snapshot.AgentType,
			RequestCount: snapshot.RequestCount,
			SuccessCount: snapshot.SuccessCount,
			LatencySumMs: snapshot.LatencySumMs,
			LatencyP50Ms: snapshot.LatencyP50Ms,
			LatencyP95Ms: snapshot.LatencyP95Ms,
			Errors:       "{}",
		})
		if err != nil {
			slog.Error("failed to persist agent metrics",
				"agent_type", snapshot.AgentType,
				"hour", snapshot.HourBucket,
				"error", err,
			)
		}
	}

	// Flush tool metrics
	toolSnapshots := p.aggregator.FlushToolMetrics(currentHour)
	for _, snapshot := range toolSnapshots {
		_, err := p.store.UpsertToolMetrics(ctx, &store.UpsertToolMetrics{
			HourBucket:   snapshot.HourBucket,
			ToolName:     snapshot.ToolName,
			CallCount:    snapshot.CallCount,
			SuccessCount: snapshot.SuccessCount,
			LatencySumMs: snapshot.LatencySumMs,
		})
		if err != nil {
			slog.Error("failed to persist tool metrics",
				"tool_name", snapshot.ToolName,
				"hour", snapshot.HourBucket,
				"error", err,
			)
		}
	}

	return nil
}

func (p *Persister) flushLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			// Final flush before shutdown
			_ = p.Flush(context.Background())
			return
		case <-ticker.C:
			if err := p.Flush(p.ctx); err != nil {
				slog.Error("periodic metrics flush failed", "error", err)
			}
		}
	}
}

func (p *Persister) cleanupLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.cleanup()
		}
	}
}

func (p *Persister) cleanup() {
	cutoff := time.Now().Add(-p.retentionPeriod)

	if err := p.store.DeleteAgentMetrics(p.ctx, &store.DeleteAgentMetrics{
		BeforeTime: &cutoff,
	}); err != nil {
		slog.Error("failed to cleanup old agent metrics", "error", err)
	}

	if err := p.store.DeleteToolMetrics(p.ctx, &store.DeleteToolMetrics{
		BeforeTime: &cutoff,
	}); err != nil {
		slog.Error("failed to cleanup old tool metrics", "error", err)
	}

	slog.Debug("metrics cleanup completed", "cutoff", cutoff)
}
