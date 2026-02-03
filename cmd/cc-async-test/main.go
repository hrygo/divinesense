package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/hrygo/divinesense/ai/agent"
)

func main() {
	// Debug level to see everything
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	runner, err := agent.NewCCRunner(10*time.Minute, logger)
	if err != nil {
		panic(err)
	}

	// Generate a new UUID for each test run to avoid session conflicts
	// 每次测试运行生成新的 UUID 以避免会话冲突
	sessionID := uuid.New().String()
	workDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get working directory: %v", err))
	}

	cfg := &agent.CCRunnerConfig{
		Mode:           "geek",
		WorkDir:        workDir,
		SessionID:      sessionID,
		UserID:         1001,
		PermissionMode: "bypassPermissions", // Ensure tools run
	}

	logger.Info("Starting Exhaustive Event Test...", "session_id", sessionID)

	sess, err := runner.StartAsyncSession(context.Background(), cfg)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := runner.GetSessionManager().TerminateSession(sessionID); err != nil {
			logger.Warn("Failed to terminate session", "error", err)
		}
	}()

	streamer := agent.NewBiDirectionalStreamer(logger)
	eventChan := make(chan agent.StreamEvent, 100)

	// Capture Stderr
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := sess.Stderr.Read(buf)
			if n > 0 {
				fmt.Printf("\n[STDERR] %s\n", buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		err := streamer.StreamOutput(sess.Stdout, eventChan)
		if err != nil {
			logger.Error("Stream ended", "error", err)
		}
	}()

	// Event Collector
	go func() {
		for evt := range eventChan {
			// Print nicely formatted JSON for inspection
			data, err := json.MarshalIndent(evt, "", "  ")
			if err != nil {
				logger.Error("Failed to marshal event", "error", err)
				continue
			}
			fmt.Printf("\n[CAPTURED EVENT] Type: %s\n%s\n", evt.Type, string(data))
		}
	}()

	// Orchestrate Complex Interactions
	interactions := []string{
		// 1. Thinking & Basic Answer
		"Please verify 1+1. Show your thinking process first.",

		// 2. File Creation (Tool: Write)
		"Create a file named 'event_test.txt' with content 'Event Type Discovery'.",

		// 3. Command Execution (Tool: Run) & Error Handling (stderr/exit code)
		"Run command 'cat event_test.txt' and then run 'ls_non_existent_command' to generate an error.",

		// 4. Cleanup
		"Remove 'event_test.txt'.",
	}

	for i, prompt := range interactions {
		time.Sleep(5 * time.Second) // Give time for previous to settle
		logger.Info(">>> Sending Prompt", "step", i+1, "prompt", prompt)

		inputMsg := streamer.BuildUserMessage(prompt)
		if err := sess.WriteInput(inputMsg); err != nil {
			panic(err)
		}

		// Wait longer for execution
		time.Sleep(15 * time.Second)
	}

	logger.Info("Test Completed. Please review logs.")
	time.Sleep(2 * time.Second)
}
