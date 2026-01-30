package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GitHubService provides GitHub API integration for PR creation.
// GitHubService 提供 GitHub API 集成以创建 PR。
type GitHubService struct {
	token           string
	repositoryOwner string
	repositoryName  string
	client          *http.Client
	baseURL         string
}

// CreatePRRequest holds parameters for creating a pull request.
// CreatePRRequest 保存创建拉取请求的参数。
type CreatePRRequest struct {
	HeadBranch string // The branch containing your changes (e.g., "evolution/task-123")
	BaseBranch string // The branch to merge into (usually "main")
	Title      string // PR title
	Body       string // PR description
}

// CreatePRResponse holds the response from creating a PR.
// CreatePRResponse 保存创建 PR 的响应。
type CreatePRResponse struct {
	URL    string // PR URL
	Number int    // PR number
	State  string // PR state
}

// NewGitHubService creates a new GitHubService instance.
// NewGitHubService 创建一个新的 GitHubService 实例。
func NewGitHubService(token, owner, name string) *GitHubService {
	return &GitHubService{
		token:           token,
		repositoryOwner: owner,
		repositoryName:  name,
		client:          &http.Client{},
		baseURL:         "https://api.github.com",
	}
}

// CreatePR creates a pull request on GitHub.
// CreatePR 在 GitHub 上创建拉取请求。
func (s *GitHubService) CreatePR(ctx context.Context, req *CreatePRRequest) (string, error) {
	if s.token == "" {
		return "", fmt.Errorf("GitHub token is required for PR creation")
	}

	// Build PR request body
	prBody := map[string]interface{}{
		"title": req.Title,
		"head":  req.HeadBranch,
		"base":  req.BaseBranch,
		"body":  req.Body,
	}

	jsonBody, err := json.Marshal(prBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal PR request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("%s/repos/%s/%s/pulls",
		s.baseURL,
		s.repositoryOwner,
		s.repositoryName)

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var prResp struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
		State   string `json:"state"`
	}
	if err := json.Unmarshal(body, &prResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return prResp.HTMLURL, nil
}

// GetPR retrieves information about an existing pull request.
// GetPR 获取现有拉取请求的信息。
func (s *GitHubService) GetPR(ctx context.Context, prNumber int) (*CreatePRResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d",
		s.baseURL,
		s.repositoryOwner,
		s.repositoryName,
		prNumber)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var prResp struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
		State   string `json:"state"`
	}
	if err := json.Unmarshal(body, &prResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &CreatePRResponse{
		URL:    prResp.HTMLURL,
		Number: prResp.Number,
		State:  prResp.State,
	}, nil
}

// ListOpenPRs lists all open pull requests.
// ListOpenPRs 列出所有开放的拉取请求。
func (s *GitHubService) ListOpenPRs(ctx context.Context) ([]CreatePRResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls?state=open",
		s.baseURL,
		s.repositoryOwner,
		s.repositoryName)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var prs []struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
		State   string `json:"state"`
	}
	if err := json.Unmarshal(body, &prs); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := make([]CreatePRResponse, len(prs))
	for i, pr := range prs {
		result[i] = CreatePRResponse{
			URL:    pr.HTMLURL,
			Number: pr.Number,
			State:  pr.State,
		}
	}

	return result, nil
}
