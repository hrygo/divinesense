// Package stats provides async persistence for session statistics.
package stats

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/store"
)

const (
	duplicateSessionWindow = 5 * time.Second // Window for duplicate detection
)

// Persister handles async persistence of session statistics.
// Persister 处理会话统计数据的异步持久化。
type Persister struct {
	store        store.AgentStatsStore
	queue        chan *agent.AgentSessionStatsForStorage
	wg           sync.WaitGroup
	logger       *slog.Logger
	stopCh       chan struct{}
	once         sync.Once
	seenSessions sync.Map // map[string]int64 - session ID -> last enqueued timestamp
	dedupEnabled atomic.Bool
}

// NewPersister creates a new async persister.
// NewPersister 创建一个新的异步持久化器。
func NewPersister(store store.AgentStatsStore, queueSize int, logger *slog.Logger) *Persister {
	if logger == nil {
		logger = slog.Default()
	}

	p := &Persister{
		store:  store,
		queue:  make(chan *agent.AgentSessionStatsForStorage, queueSize),
		logger: logger,
		stopCh: make(chan struct{}),
	}
	p.dedupEnabled.Store(true) // Enable deduplication by default
	p.wg.Add(1)
	go p.processQueue()
	return p
}

// Enqueue queues a stats record for persistence.
// Enqueue 将统计记录排队等待持久化。
// Returns true if successfully queued, false if queue is full or duplicate detected.
func (p *Persister) Enqueue(stats *agent.AgentSessionStatsForStorage) bool {
	// Idempotency check: prevent duplicate enqueues within a time window
	if p.dedupEnabled.Load() {
		if lastEnqueued, ok := p.seenSessions.Load(stats.SessionID); ok {
			if lastTs, ok := lastEnqueued.(int64); ok {
				lastTime := time.Unix(lastTs, 0)
				if time.Since(lastTime) < duplicateSessionWindow {
					p.logger.Debug("Persister: ignoring duplicate stats",
						"session_id", stats.SessionID,
						"last_enqueued", lastTime,
						"elapsed_ms", time.Since(lastTime).Milliseconds())
					return false
				}
			}
		}
		// Record this session enqueue time
		p.seenSessions.Store(stats.SessionID, time.Now().Unix())
	}

	select {
	case p.queue <- stats:
		p.logger.Debug("Persister: stats enqueued",
			"session_id", stats.SessionID,
			"cost_usd", stats.TotalCostUSD,
			"queue_size", len(p.queue))
		return true
	default:
		p.logger.Warn("Persister: queue full, dropping stats record",
			"session_id", stats.SessionID,
			"queue_size", len(p.queue))
		return false
	}
}

// EnqueueSessionStatsData converts SessionStatsData and enqueues it.
func (p *Persister) EnqueueSessionStatsData(data *agent.SessionStatsData) bool {
	return p.Enqueue(data.ToAgentSessionStats())
}

// processQueue processes stats records in the background.
// processQueue 在后台处理统计记录。
func (p *Persister) processQueue() {
	defer p.wg.Done()

	for {
		select {
		case stats := <-p.queue:
			if stats == nil {
				// Nil signal means shutdown
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := p.saveSessionStats(ctx, stats)
			cancel()

			if err != nil {
				p.logger.Error("Persister: failed to save session stats",
					"session_id", stats.SessionID,
					"error", err)
			} else {
				p.logger.Debug("Persister: saved session stats",
					"session_id", stats.SessionID,
					"cost_usd", stats.TotalCostUSD)
			}

		case <-p.stopCh:
			// Drain remaining items before shutdown
			p.drainQueue()
			return
		}
	}
}

// saveSessionStats converts and saves session stats to the database.
func (p *Persister) saveSessionStats(ctx context.Context, stats *agent.AgentSessionStatsForStorage) error {
	// Convert to store format
	storeStats := &store.AgentSessionStats{
		SessionID:            stats.SessionID,
		ConversationID:       stats.ConversationID,
		UserID:               stats.UserID,
		AgentType:            stats.AgentType,
		StartedAt:            stats.StartTime,
		EndedAt:              stats.EndedAt,
		TotalDurationMs:      stats.TotalDurationMs,
		ThinkingDurationMs:   stats.ThinkingDurationMs,
		ToolDurationMs:       stats.ToolDurationMs,
		GenerationDurationMs: stats.GenerationDurationMs,
		InputTokens:          stats.InputTokens,
		OutputTokens:         stats.OutputTokens,
		CacheWriteTokens:     stats.CacheWriteTokens,
		CacheReadTokens:      stats.CacheReadTokens,
		TotalTokens:          stats.TotalTokens,
		TotalCostUSD:         stats.TotalCostUSD,
		ToolCallCount:        stats.ToolCallCount,
		ToolsUsed:            stats.ToolsUsed,
		FilesModified:        stats.FilesModified,
		FilePaths:            stats.FilePaths,
		ModelUsed:            stats.ModelUsed,
		IsError:              stats.IsError,
		ErrorMessage:         stats.ErrorMessage,
	}

	return p.store.SaveSessionStats(ctx, storeStats)
}

// drainQueue processes any remaining items in the queue during shutdown.
// drainQueue 在关闭期间处理队列中剩余的项。
func (p *Persister) drainQueue() {
	p.logger.Info("Persister: draining queue", "remaining", len(p.queue))
	lostCount := 0
	savedCount := 0
	for {
		select {
		case stats := <-p.queue:
			if stats == nil {
				if lostCount > 0 {
					p.logger.Error("Persister: shutdown complete with data loss",
						"saved", savedCount,
						"lost", lostCount)
				}
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := p.saveSessionStats(ctx, stats)
			cancel()

			if err != nil {
				lostCount++
				p.logger.Error("Persister: failed to save session stats during shutdown",
					"session_id", stats.SessionID,
					"cost_usd", stats.TotalCostUSD,
					"error", err)
			} else {
				savedCount++
			}

		default:
			if lostCount > 0 {
				p.logger.Error("Persister: shutdown complete with data loss",
					"saved", savedCount,
					"lost", lostCount)
			}
			return
		}
	}
}

// Close waits for the queue to drain and shuts down the persister.
// Close 等待队列清空并关闭持久化器。
func (p *Persister) Close(timeout time.Duration) error {
	p.once.Do(func() {
		close(p.stopCh)
	})

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("Persister: shutdown complete")
		return nil
	case <-time.After(timeout):
		p.logger.Warn("Persister: shutdown timeout")
		return context.DeadlineExceeded
	}
}

// QueueSize returns the current queue size.
// QueueSize 返回当前队列大小。
func (p *Persister) QueueSize() int {
	return len(p.queue)
}
