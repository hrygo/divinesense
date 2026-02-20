//go:build ignore
// +build ignore

// test_multiplex_prompt is a manual integration test for CCRunner multiplexing.
// It requires a running Claude CLI and is NOT executed during CI (go test ./...).
//
// Usage:
//
//	go run scripts/test_multiplex_prompt.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hrygo/divinesense/ai/agents/geek"
	"github.com/hrygo/divinesense/ai/agents/runner"

	agentpkg "github.com/hrygo/divinesense/ai/agents"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	sm := runner.NewCCSessionManager(nil, 5*time.Minute)
	defer sm.Shutdown()

	sessionID := uuid.New().String()
	userID := int32(1)

	// Build WorkDir using GeekMode logic
	geekMode := geek.NewGeekMode("")
	workDir := geekMode.GetWorkDir(userID)

	// Ensure workDir exists
	if err := os.MkdirAll(workDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create workDir %s: %v", workDir, err)
	}

	// Build the full Geek Mode system prompt
	ccCfg := &agentpkg.CCRunnerConfig{
		WorkDir:       workDir,
		SessionID:     sessionID,
		UserID:        userID,
		DeviceContext: `{"userAgent":"Go-Test/1.0","isMobile":false,"screenWidth":1920,"screenHeight":1080,"language":"zh-CN"}`,
	}
	systemPrompt := geekMode.BuildSystemPrompt(ccCfg)

	log.Printf("📋 System Prompt (%d bytes):\n%s\n---", len(systemPrompt), systemPrompt)

	cfg := runner.Config{
		WorkDir:        workDir,
		PermissionMode: "bypassPermissions",
		SystemPrompt:   systemPrompt,
	}

	sess, err := sm.GetOrCreateSession(context.Background(), sessionID, cfg)
	if err != nil {
		log.Fatalf("❌ GetOrCreateSession failed: %v", err)
	}

	log.Printf("✅ Session created: %s (workDir: %s)", sessionID, workDir)

	// Wait for CLI to boot
	time.Sleep(3 * time.Second)

	prompt := "列出当前 user（id=1） 目录下有哪些文件"
	log.Printf("📤 Sending prompt: %s", prompt)

	done := make(chan struct{})
	var mu sync.Mutex
	var events []string

	cb := func(eventType string, data any) error {
		mu.Lock()
		defer mu.Unlock()

		switch eventType {
		case "system":
			// skip system events for cleaner output
		case "assistant":
			if m, ok := data.(runner.StreamMessage); ok {
				raw, _ := json.Marshal(m)
				log.Printf("🤖 [assistant] %s", string(raw))
			}
		case "result":
			if m, ok := data.(runner.StreamMessage); ok {
				raw, _ := json.MarshalIndent(m, "", "  ")
				fmt.Printf("\n--- Result Event ---\n%s\n", string(raw))
			}
			log.Printf("✅ [result] turn complete")
		default:
			log.Printf("📨 [%s] event", eventType)
		}

		events = append(events, eventType)
		return nil
	}

	sess.SetCallback(cb, done)

	msg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role": "user",
			"content": []map[string]any{
				{"type": "text", "text": prompt},
			},
		},
	}

	if err := sess.WriteInput(msg); err != nil {
		log.Fatalf("❌ WriteInput failed: %v", err)
	}

	select {
	case <-done:
		mu.Lock()
		log.Printf("✅ Turn completed with %d events", len(events))
		mu.Unlock()
	case <-time.After(180 * time.Second):
		log.Fatal("❌ Timeout after 180s!")
	}

	time.Sleep(500 * time.Millisecond)
	log.Println("🏁 Test finished. Shutting down...")
	os.Exit(0)
}
