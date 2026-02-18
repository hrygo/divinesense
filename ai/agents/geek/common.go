package geek

import (
	"context"
	"fmt"
	"time"

	agentpkg "github.com/hrygo/divinesense/ai/agents"
)

// ExecutePersistentSession executes a command using a persistent session.
// ExecutePersistentSession 使用持久化会话执行命令。
// It handles:
// 1. Session start/retrieval via StartAsyncSession
// 2. Callback wrapping to maximize responsiveness (streaming answer/result)
// 3. Stats collection
// 4. Input writing
// 5. Completion signal waiting
func ExecutePersistentSession(
	ctx context.Context,
	r *agentpkg.CCRunner, // Interface or struct? It's *runner.CCRunner
	cfg *agentpkg.CCRunnerConfig,
	input map[string]any,
	callback agentpkg.EventCallback,
) error {
	// Start or Retrieve persistent session
	session, err := r.StartAsyncSession(ctx, cfg)
	if err != nil {
		return fmt.Errorf("StartAsyncSession failed: %w", err)
	}

	// Wait for CLI to be ready (init event received)
	// 等待 CLI 就绪（收到 init 事件）
	readyCtx, readyCancel := context.WithTimeout(ctx, 30*time.Second)
	defer readyCancel()
	if err := session.WaitForReady(readyCtx); err != nil {
		return fmt.Errorf("WaitForReady failed: %w", err)
	}

	// Wait channel for turn completion
	done := make(chan error, 1)

	// Wrap callback to intercept completion signals
	wrappedCallback := func(event string, data any) error {
		if event == "tool_result" || event == "answer" {
			// Pass through content events
			if err := callback(event, data); err != nil {
				return err
			}
		} else if event == "result" {
			// Turn completed successfully
			select {
			case done <- nil:
			default:
			}
			return nil
		} else if event == "error" {
			// Turn failed
			// Extract error details if possible
			select {
			case done <- fmt.Errorf("session error"):
			default:
			}
			return nil
		} else {
			// Forward other events (tool_use, thinking, etc)
			if err := callback(event, data); err != nil {
				return err
			}
		}
		return nil
	}

	// Create stats for this turn
	turnStats := &agentpkg.SessionStats{
		SessionID: cfg.SessionID,
		StartTime: time.Now(),
		ToolsUsed: make(map[string]bool),
	}

	// Attach callback and stats to session
	session.SetCallback(wrappedCallback)
	session.SetStats(turnStats)

	// Ensure cleanup on exit
	defer func() {
		session.SetCallback(nil)
		session.SetStats(nil)
	}()

	// Send user input
	if err := session.WriteInput(input); err != nil {
		return fmt.Errorf("WriteInput failed: %w", err)
	}

	// Wait for completion
	select {
	case err := <-done:
		if err != nil {
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
