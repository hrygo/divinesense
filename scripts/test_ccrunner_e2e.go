//go:build ignore
// +build ignore

// test_ccrunner_e2e is a manual end-to-end test for the integrated CCRunner.
// It exercises the full pipeline: Execute() → executeWithMultiplex → SessionManager.
// NOT executed during CI (go test ./...).
//
// Usage:
//
//	go run scripts/test_ccrunner_e2e.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hrygo/divinesense/ai/agents/geek"
	"github.com/hrygo/divinesense/ai/agents/runner"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	r, err := runner.NewCCRunner(3*time.Minute, logger)
	if err != nil {
		log.Fatalf("❌ NewCCRunner failed: %v", err)
	}

	userID := int32(1)
	geekMode := geek.NewGeekMode("")
	workDir := geekMode.GetWorkDir(userID)

	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("❌ MkdirAll failed: %v", err)
	}

	// Build Geek mode system prompt
	ccCfg := &runner.Config{
		WorkDir:       workDir,
		SessionID:     "test-e2e-session",
		UserID:        userID,
		DeviceContext: `{"userAgent":"E2E-Test/1.0","isMobile":false,"language":"zh-CN"}`,
	}
	systemPrompt := geekMode.BuildSystemPrompt(ccCfg)

	cfg := &runner.Config{
		Mode:           "geek",
		WorkDir:        workDir,
		ConversationID: 99999, // will be mapped to UUID v5 SessionID
		UserID:         userID,
		SystemPrompt:   systemPrompt,
		PermissionMode: "bypassPermissions",
	}

	// ─── Define test turns ─────────────────────────────────────────────
	turns := []struct {
		Name   string
		Prompt string
	}{
		{
			Name:   "Turn 1: 复杂多步骤指令",
			Prompt: "请完成以下三步操作并汇总结果：1) 用 bash 查看当前工作目录下的文件列表；2) 创建一个名为 test_report.md 的 Markdown 文件，内容包含当前时间和系统信息；3) 确认文件已创建并报告文件大小。",
		},
		{
			Name:   "Turn 2: 基于上一轮结果追问（验证会话持久化）",
			Prompt: "刚才创建的 test_report.md 文件内容中，操作系统信息是什么？请直接从上下文中回答，不要重新读取文件。",
		},
	}

	// ─── Execute turns ─────────────────────────────────────────────────
	for i, turn := range turns {
		log.Printf("\n" + strings.Repeat("═", 60))
		log.Printf("📤 %s", turn.Name)
		log.Printf("   Prompt: %s", turn.Prompt)
		log.Printf(strings.Repeat("═", 60) + "\n")

		var mu sync.Mutex
		var eventTypes []string
		var lastResult string
		turnStart := time.Now()

		callback := func(eventType string, data any) error {
			mu.Lock()
			defer mu.Unlock()
			eventTypes = append(eventTypes, eventType)

			switch eventType {
			case "thinking":
				// skip for cleaner output
			case "answer":
				if s, ok := data.(string); ok {
					log.Printf("   📝 [answer] %s", truncate(s, 120))
				}
			case "session_stats":
				raw, _ := json.MarshalIndent(data, "   ", "  ")
				fmt.Printf("\n   --- Session Stats ---\n   %s\n", string(raw))
			case "error":
				log.Printf("   ❌ [error] %v", data)
			default:
				// Log assistant/tool events
				if m, ok := data.(*runner.EventWithMeta); ok {
					log.Printf("   📨 [%s] %s", eventType, truncate(m.EventData, 100))
					lastResult = m.EventData
				} else {
					log.Printf("   📨 [%s] %T", eventType, data)
				}
			}
			return nil
		}

		err := r.Execute(context.Background(), cfg, turn.Prompt, callback)
		duration := time.Since(turnStart)

		mu.Lock()
		eventCount := len(eventTypes)
		mu.Unlock()

		if err != nil {
			log.Printf("❌ Turn %d failed: %v", i+1, err)
			os.Exit(1)
		}

		log.Printf("✅ Turn %d completed: %d events, %.1fs", i+1, eventCount, duration.Seconds())
		if lastResult != "" {
			log.Printf("   Last result: %s", truncate(lastResult, 200))
		}
	}

	log.Println("\n🏁 All turns completed. Test passed!")
	os.Exit(0)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
