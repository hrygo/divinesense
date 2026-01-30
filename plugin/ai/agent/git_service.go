package agent

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// GitService provides Git operations for Evolution Mode.
// GitService 为进化模式提供 Git 操作。
type GitService struct {
	repoDir string
	mu      sync.Mutex
}

// NewGitService creates a new GitService instance.
// NewGitService 创建一个新的 GitService 实例。
func NewGitService(repoDir string) *GitService {
	return &GitService{
		repoDir: repoDir,
	}
}

// git executes a git command in the repository directory.
// git 在仓库目录中执行 git 命令。
func (s *GitService) git(args ...string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd := exec.Command("git", args...)
	cmd.Dir = s.repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\n%s", strings.Join(args, " "), err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// GetCurrentBranch returns the current branch name.
// GetCurrentBranch 返回当前分支名称。
func (s *GitService) GetCurrentBranch() (string, error) {
	branch, err := s.git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return branch, nil
}

// CreateBranch creates a new branch from the current HEAD.
// CreateBranch 从当前 HEAD 创建新分支。
func (s *GitService) CreateBranch(branchName string) error {
	_, err := s.git("checkout", "-b", branchName)
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w", branchName, err)
	}
	return nil
}

// CheckoutBranch switches to the specified branch.
// CheckoutBranch 切换到指定分支。
func (s *GitService) CheckoutBranch(branchName string) error {
	_, err := s.git("checkout", branchName)
	if err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
	}
	return nil
}

// Add stages files for commit.
// Add 暂存文件以供提交。
func (s *GitService) Add(files ...string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files specified")
	}
	_, err := s.git(append([]string{"add"}, files...)...)
	if err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}
	return nil
}

// Commit creates a commit with the given message.
// Commit 使用给定消息创建提交。
func (s *GitService) Commit(message string) error {
	_, err := s.git("commit", "-m", message)
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}
	return nil
}

// GetStatus returns the git status.
// GetStatus 返回 git 状态。
func (s *GitService) GetStatus() (*GitStatus, error) {
	output, err := s.git("status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	status := &GitStatus{
		Modified: make([]string, 0),
		Added:    make([]string, 0),
		Deleted:  make([]string, 0),
	}

	if output == "" {
		return status, nil
	}

	// Don't trim again - git() already trimmed, and we need to split by newline
	// If output is just whitespace or empty after split, skip
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		statusCode := line[:2]
		// File path starts after the 2-character status code
		// Use TrimSpace to handle the space separator and any trailing whitespace
		filePath := strings.TrimSpace(line[2:])

		switch statusCode {
		case "M", "MM", " M":
			status.Modified = append(status.Modified, filePath)
		case "A", "AM", "M ": // Staged modifications also count as added
			status.Added = append(status.Added, filePath)
		case "D", "DD":
			status.Deleted = append(status.Deleted, filePath)
		case "??", "? ":
			// Untracked files - add to Added for our purposes
			status.Added = append(status.Added, filePath)
		}
	}

	return status, nil
}

// HasChanges returns true if there are uncommitted changes.
// HasChanges 如果有未提交的更改则返回 true。
func (s *GitService) HasChanges() (bool, error) {
	status, err := s.GetStatus()
	if err != nil {
		return false, err
	}
	return len(status.Modified)+len(status.Added)+len(status.Deleted) > 0, nil
}

// CreateEvolutionBranch creates a new evolution branch with the format "evolution/{task-id}".
// CreateEvolutionBranch 创建格式为 "evolution/{task-id}" 的新进化分支。
func (s *GitService) CreateEvolutionBranch(taskID string) (string, error) {
	branchName := filepath.Join("evolution", taskID)

	// Check if branch already exists
	_, err := s.git("rev-parse", "--verify", branchName)
	if err == nil {
		// Branch exists, checkout to it
		return branchName, s.CheckoutBranch(branchName)
	}

	// Create new branch from main/develop
	currentBranch, err := s.GetCurrentBranch()
	if err != nil {
		return "", err
	}

	// Ensure we're on main or a clean state
	if currentBranch != "main" {
		// Checkout main first
		if err := s.CheckoutBranch("main"); err != nil {
			return "", fmt.Errorf("failed to checkout main: %w", err)
		}
	}

	// Create and checkout new branch
	if err := s.CreateBranch(branchName); err != nil {
		return "", fmt.Errorf("failed to create evolution branch: %w", err)
	}

	return branchName, nil
}

// CommitChanges stages and commits all changes with a conventional commit message.
// CommitChanges 使用约定式提交消息暂存并提交所有更改。
func (s *GitService) CommitChanges(message string) error {
	// Add all changes
	if err := s.Add("."); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Commit with message
	if err := s.Commit(message); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	return nil
}

// GitStatus represents the status of the git repository.
// GitStatus 表示 git 仓库的状态。
type GitStatus struct {
	Modified []string
	Added    []string
	Deleted  []string
}

// IsClean returns true if there are no changes.
func (s *GitStatus) IsClean() bool {
	return len(s.Modified)+len(s.Added)+len(s.Deleted) == 0
}

// HasFile returns true if the file is in any of the change lists.
func (s *GitStatus) HasFile(file string) bool {
	file = filepath.Clean(file)
	for _, f := range s.Modified {
		if filepath.Clean(f) == file {
			return true
		}
	}
	for _, f := range s.Added {
		if filepath.Clean(f) == file {
			return true
		}
	}
	for _, f := range s.Deleted {
		if filepath.Clean(f) == file {
			return true
		}
	}
	return false
}
