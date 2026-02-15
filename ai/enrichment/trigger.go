package enrichment

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Trigger 异步触发器
type Trigger struct {
	pipeline *Pipeline
	queue    chan *MemoContent
	workers  int
	wg       sync.WaitGroup
	stopCh   chan struct{}
}

// NewTrigger 创建新的触发器
func NewTrigger(pipeline *Pipeline, workers int) *Trigger {
	if workers <= 0 {
		workers = 3 // Default workers
	}
	return &Trigger{
		pipeline: pipeline,
		queue:    make(chan *MemoContent, 100), // Buffered queue
		workers:  workers,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动触发器
func (t *Trigger) Start() {
	for i := 0; i < t.workers; i++ {
		t.wg.Add(1)
		go t.worker(i)
	}
	slog.Info("enrichment trigger started", "workers", t.workers)
}

// Stop 停止触发器
func (t *Trigger) Stop() {
	close(t.stopCh)
	t.wg.Wait()
	close(t.queue)
	slog.Info("enrichment trigger stopped")
}

// TriggerAsync 异步触发 enrichment
// 这是一个非阻塞调用，任务会被放入队列中异步执行
func (t *Trigger) TriggerAsync(content *MemoContent) {
	select {
	case t.queue <- content:
		// Successfully queued
	case <-time.After(50 * time.Millisecond):
		// Queue is full, graceful degradation - skip this trigger
		slog.Debug("enrichment trigger skipped (queue full)", "memo_id", content.MemoID)
	case <-t.stopCh:
		// Trigger is stopped
	}
}

// worker 处理队列中的任务
func (t *Trigger) worker(id int) {
	defer t.wg.Done()

	for {
		select {
		case <-t.stopCh:
			return
		case content, ok := <-t.queue:
			if !ok {
				return
			}
			t.processContent(content, id)
		}
	}
}

// processContent 处理单个 memo 内容
func (t *Trigger) processContent(content *MemoContent, workerID int) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	slog.Debug("enrichment trigger processing",
		"memo_id", content.MemoID,
		"worker", workerID)

	// Execute post-save pipeline (tags, title, summary)
	results := t.pipeline.EnrichPostSave(ctx, content)

	// Log results (skip if no post-enrichers configured)
	if results == nil {
		slog.Debug("enrichment trigger skipped (no post-enrichers)",
			"memo_id", content.MemoID,
			"worker", workerID)
		return
	}
	for _, result := range results {
		status := "success"
		if !result.Success {
			status = "failed"
		}
		slog.Debug("enrichment result",
			"type", result.Type,
			"status", status,
			"latency_ms", result.Latency.Milliseconds(),
			"memo_id", content.MemoID,
			"worker", workerID)
	}

	slog.Debug("enrichment trigger completed",
		"memo_id", content.MemoID,
		"worker", workerID,
		"total_latency_ms", time.Since(start).Milliseconds())
}
