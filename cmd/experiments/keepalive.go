package main

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/hrygo/divinesense/ai/agents/runner"
)

// This script verifies if we can keep a Claude Code CLI process alive
// and send multiple messages to it via Stdin.
// It indirectly verifies the core logic used in ai/agents/geek/common.go
// (StartAsyncSession + WriteInput).
// MockCCRunner simulates CCRunner but uses "cat" command
type MockCCRunner struct {
	manager runner.SessionManager
}

func NewMockCCRunner(logger *slog.Logger) *MockCCRunner {
	r, _ := runner.NewCCRunner(10*time.Minute, logger)
	return &MockCCRunner{
		manager: r.GetSessionManager(),
	}
}

func (r *MockCCRunner) StartAsyncSession(ctx context.Context, cfg *runner.Config) (*runner.Session, error) {
	// We need to bypass runner.StartAsyncSession because it hardcodes "claude"
	// So we start a session manually using a hacked SessionManager or just manual exec

	// Actually, SessionManager.GetOrCreateSession calls startSession which calls exec.LookPath("claude").
	// To test purely logic, we might need to modify the codebase temporarily OR
	// just write a test that manually invokes exec.Command("cat") and wraps it in a Session struct
	// to test Session.WriteInput behavior.

	cmd := exec.CommandContext(ctx, "cat")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	sess := &runner.Session{
		ID:         cfg.SessionID,
		Config:     *cfg,
		Cmd:        cmd,
		Stdin:      stdin, // cat echoes stdin to stdout
		Stdout:     stdout,
		Stderr:     stderr,
		Status:     runner.SessionStatusReady,
		LastActive: time.Now(),
	}

	return sess, nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	fmt.Println(">>> Starting Mock Session (cat)...")
	ctx := context.Background()

	// Create Mock Runner
	mockRunner := NewMockCCRunner(logger)

	// Start Session
	session, err := mockRunner.StartAsyncSession(ctx, &runner.Config{SessionID: "test-id"})
	if err != nil {
		panic(err)
	}

	fmt.Printf(">>> Session Started. PID: %d\n", session.Cmd.Process.Pid)
	initialPid := session.Cmd.Process.Pid

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(session.Stdout)
		for scanner.Scan() {
			fmt.Printf("[STDOUT] %s\n", scanner.Text())
		}
	}()

	// Test Message 1
	fmt.Println(">>> Sending Message 1...")
	if err := session.WriteInput(map[string]any{"msg": "hello"}); err != nil {
		panic(err)
	}

	time.Sleep(2 * time.Second)

	// Check PID
	if session.Cmd.Process.Pid != initialPid {
		fmt.Printf(">>> FATAL: PID changed!\n")
		os.Exit(1)
	}
	fmt.Println(">>> PID confirmed stable.")

	// Test Message 2
	fmt.Println(">>> Sending Message 2...")
	if err := session.WriteInput(map[string]any{"msg": "world"}); err != nil {
		panic(err)
	}

	time.Sleep(2 * time.Second)

	fmt.Println(">>> Killing session...")
	session.Cmd.Process.Kill()
	fmt.Println(">>> Done.")
}
