package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitService_CreateEvolutionBranch tests branch creation.
func TestGitService_CreateEvolutionBranch(t *testing.T) {
	// Skip if git not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Helper to run git commands in tmpDir
	runGit := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		return cmd.Run()
	}

	// Initialize a git repository
	if err := runGit("init"); err != nil {
		t.Fatalf("Failed to init git: %v", err)
	}

	// Configure git
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")

	// Create service
	git := NewGitService(tmpDir)

	// Create an initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content"), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if err := git.Add("test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if err := git.Commit("Initial commit"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test branch creation
	taskID := "task-test-001"
	branchName, err := git.CreateEvolutionBranch(taskID)
	if err != nil {
		t.Fatalf("CreateEvolutionBranch failed: %v", err)
	}

	expectedPrefix := "evolution"
	if len(branchName) < len(expectedPrefix) {
		t.Errorf("Branch name %q too short, expected prefix %q", branchName, expectedPrefix)
	}

	// Verify branch was created
	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != branchName {
		t.Errorf("Expected branch %q, got %q", branchName, currentBranch)
	}
}

// TestGitService_CommitChanges tests commit functionality.
func TestGitService_CommitChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Helper to run git commands in tmpDir
	runGit := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		return cmd.Run()
	}

	// Initialize git
	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")

	git := NewGitService(tmpDir)

	// Setup initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0o644)
	git.Add("test.txt")
	git.Commit("Initial")

	// Test committing changes
	os.WriteFile(testFile, []byte("modified content"), 0o644)

	commitMsg := "feat: test commit"
	if err := git.CommitChanges(commitMsg); err != nil {
		t.Fatalf("CommitChanges failed: %v", err)
	}

	// Verify no more changes
	hasChanges, _ := git.HasChanges()
	if hasChanges {
		t.Error("Expected no changes after commit")
	}
}

// TestGitService_GetStatus tests status retrieval.
func TestGitService_GetStatus(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Helper to run git commands in tmpDir
	runGit := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		return cmd.Run()
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")

	git := NewGitService(tmpDir)

	status, err := git.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	// Clean repo should have empty status
	if !status.IsClean() {
		t.Error("Expected clean status for new repo")
	}
}

// TestGitService_CheckoutBranch tests checkout functionality.
func TestGitService_CheckoutBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Helper to run git commands in tmpDir
	runGit := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		return cmd.Run()
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")

	git := NewGitService(tmpDir)

	// Create initial commit on main
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("main content"), 0o644)
	git.Add("test.txt")
	git.Commit("Initial commit")

	// Create a new branch
	_, err = git.CreateEvolutionBranch("test-task")
	if err != nil {
		t.Fatalf("CreateEvolutionBranch failed: %v", err)
	}

	// Modify file on new branch
	os.WriteFile(testFile, []byte("branch content"), 0o644)
	git.Add("test.txt")
	git.Commit("Branch commit")

	// Checkout back to main
	if err := git.CheckoutBranch("main"); err != nil {
		t.Fatalf("CheckoutBranch to main failed: %v", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	if currentBranch != "main" {
		t.Errorf("Expected branch 'main', got %q", currentBranch)
	}
}

// TestGitService_HasChanges tests change detection.
func TestGitService_HasChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Helper to run git commands in tmpDir
	runGit := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		return cmd.Run()
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")

	git := NewGitService(tmpDir)

	// No changes initially (empty repo)
	hasChanges, err := git.HasChanges()
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if hasChanges {
		t.Error("Expected no changes for empty repo")
	}

	// Create initial commit to establish tracking
	testFile := filepath.Join(tmpDir, "initial.txt")
	os.WriteFile(testFile, []byte("initial content"), 0o644)
	git.Add("initial.txt")
	git.Commit("Initial commit")

	// Now modify the file - should have changes
	os.WriteFile(testFile, []byte("modified content"), 0o644)

	// Debug: check raw git status output
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = tmpDir
	rawOutput, _ := cmd.Output()
	rawStr := string(rawOutput)
	t.Logf("Raw git status: %q (len=%d)", rawStr, len(rawStr))

	// Debug: show each line's status code
	lines := strings.Split(rawStr, "\n")
	for i, line := range lines {
		if len(line) >= 2 {
			t.Logf("Line %d: %q (statusCode=%q)", i, line, line[:2])
		}
	}

	status, err := git.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	t.Logf("GetStatus result - Modified: %v, Added: %v, Deleted: %v (IsClean=%v)", status.Modified, status.Added, status.Deleted, status.IsClean())

	// Also check HasChanges directly
	hasChangesDirect, err := git.HasChanges()
	t.Logf("HasChanges directly: %v (err=%v)", hasChangesDirect, err)

	hasChanges, err = git.HasChanges()
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("Expected changes after modifying tracked file")
	}

	// Add the change - still has staged changes
	git.Add("initial.txt")

	status, err = git.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	t.Logf("After add - Modified: %v, Added: %v, Deleted: %v", status.Modified, status.Added, status.Deleted)

	hasChanges, err = git.HasChanges()
	if err != nil {
		t.Fatalf("HasChanges failed: %v", err)
	}
	if !hasChanges {
		t.Error("Expected staged changes after adding file")
	}
}

// TestGitStatus_HasFile tests file status checking.
func TestGitStatus_HasFile(t *testing.T) {
	status := &GitStatus{
		Modified: []string{"file1.go", "file2.go"},
		Added:    []string{"file3.go"},
		Deleted:  []string{"file4.go"},
	}

	tests := []struct {
		file    string
		want    bool
	}{
		{"file1.go", true},
		{"file2.go", true},
		{"file3.go", true},
		{"file4.go", true},
		{"file5.go", false},
		{"missing.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got := status.HasFile(tt.file)
			if got != tt.want {
				t.Errorf("HasFile(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

// TestGitService_Add_EmptyFiles tests adding with no files.
func TestGitService_Add_EmptyFiles(t *testing.T) {
	git := NewGitService("/tmp")
	err := git.Add()
	if err == nil {
		t.Error("Expected error for Add with no files, got nil")
	}
}
