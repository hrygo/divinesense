package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGitHubService_CreatePR tests PR creation.
func TestGitHubService_CreatePR(t *testing.T) {
	// Mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify URL path
		if !strings.Contains(r.URL.Path, "/pulls") {
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}

		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if !strings.Contains(auth, "Bearer") {
			t.Error("Expected Bearer token in Authorization header")
		}

		// Write mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"id": 1,
			"number": 123,
			"title": "Test PR",
			"state": "open",
			"html_url": "https://github.com/test/repo/pull/123"
		}`))
	}))
	defer server.Close()

	// Create GitHub service with mock server URL
	service := &GitHubService{
		client:          server.Client(),
		baseURL:         server.URL,
		token:           "test-token",
		repositoryOwner: "test",
		repositoryName:  "repo",
	}

	ctx := context.Background()
	req := &CreatePRRequest{
		HeadBranch: "evolution/test-branch",
		BaseBranch: "main",
		Title:      "Test PR",
		Body:       "Test body",
	}

	url, err := service.CreatePR(ctx, req)
	if err != nil {
		t.Fatalf("CreatePR failed: %v", err)
	}

	if !strings.Contains(url, "pull/123") {
		t.Errorf("Expected PR URL containing 'pull/123', got %s", url)
	}
}

// TestGitHubService_CreatePR_NoToken tests PR creation without token.
func TestGitHubService_CreatePR_NoToken(t *testing.T) {
	service := &GitHubService{
		token:           "",
		repositoryOwner: "test",
		repositoryName:  "repo",
	}

	ctx := context.Background()
	req := &CreatePRRequest{
		HeadBranch: "evolution/test",
		BaseBranch: "main",
		Title:      "Test",
		Body:       "Body",
	}

	_, err := service.CreatePR(ctx, req)
	if err == nil {
		t.Error("Expected error for missing token, got nil")
	}

	if !strings.Contains(err.Error(), "token") {
		t.Errorf("Expected token error, got: %v", err)
	}
}

// TestGitHubService_CreatePR_Conflict tests PR creation with conflict.
func TestGitHubService_CreatePR_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate PR already exists
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{
			"message": "A pull request already exists"
		}`))
	}))
	defer server.Close()

	service := &GitHubService{
		client:          server.Client(),
		baseURL:         server.URL,
		token:           "test-token",
		repositoryOwner: "test",
		repositoryName:  "repo",
	}

	ctx := context.Background()
	req := &CreatePRRequest{
		HeadBranch: "evolution/test",
		BaseBranch: "main",
		Title:      "Test",
		Body:       "Body",
	}

	_, err := service.CreatePR(ctx, req)
	if err == nil {
		t.Error("Expected error for PR conflict, got nil")
	}
}

// TestGitHubService_GetPR tests getting a specific PR.
func TestGitHubService_GetPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"id": 1,
			"number": 123,
			"title": "Test PR",
			"state": "open",
			"html_url": "https://github.com/test/repo/pull/123"
		}`))
	}))
	defer server.Close()

	service := &GitHubService{
		client:          server.Client(),
		baseURL:         server.URL,
		token:           "test-token",
		repositoryOwner: "test",
		repositoryName:  "repo",
	}

	ctx := context.Background()
	pr, err := service.GetPR(ctx, 123)
	if err != nil {
		t.Fatalf("GetPR failed: %v", err)
	}

	if pr.Number != 123 {
		t.Errorf("Expected PR number 123, got %d", pr.Number)
	}

	if pr.State != "open" {
		t.Errorf("Expected PR state 'open', got %s", pr.State)
	}
}

// TestGitHubService_ListOpenPRs tests listing open PRs.
func TestGitHubService_ListOpenPRs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Check for state=open query parameter
		if !strings.Contains(r.URL.String(), "state=open") {
			t.Errorf("Expected state=open in URL, got %s", r.URL.String())
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{
				"id": 1,
				"number": 100,
				"title": "First PR",
				"state": "open",
				"html_url": "https://github.com/test/repo/pull/100"
			},
			{
				"id": 2,
				"number": 101,
				"title": "Second PR",
				"state": "open",
				"html_url": "https://github.com/test/repo/pull/101"
			}
		]`))
	}))
	defer server.Close()

	service := &GitHubService{
		client:          server.Client(),
		baseURL:         server.URL,
		token:           "test-token",
		repositoryOwner: "test",
		repositoryName:  "repo",
	}

	ctx := context.Background()
	prs, err := service.ListOpenPRs(ctx)
	if err != nil {
		t.Fatalf("ListOpenPRs failed: %v", err)
	}

	if len(prs) != 2 {
		t.Errorf("Expected 2 PRs, got %d", len(prs))
	}
}

// TestGitHubService_NewGitHubService tests service constructor.
func TestGitHubService_NewGitHubService(t *testing.T) {
	service := NewGitHubService("test-token", "owner", "repo")

	if service.token != "test-token" {
		t.Errorf("Expected token 'test-token', got %s", service.token)
	}

	if service.repositoryOwner != "owner" {
		t.Errorf("Expected owner 'owner', got %s", service.repositoryOwner)
	}

	if service.repositoryName != "repo" {
		t.Errorf("Expected repo 'repo', got %s", service.repositoryName)
	}

	if service.baseURL != "https://api.github.com" {
		t.Errorf("Expected default base URL, got %s", service.baseURL)
	}

	if service.client == nil {
		t.Error("Expected client to be initialized")
	}
}

// TestCreatePRRequest tests the request structure.
func TestCreatePRRequest(t *testing.T) {
	req := &CreatePRRequest{
		HeadBranch: "evolution/test",
		BaseBranch: "main",
	}

	if req.HeadBranch != "evolution/test" {
		t.Errorf("Unexpected HeadBranch: %s", req.HeadBranch)
	}

	if req.BaseBranch != "main" {
		t.Errorf("Unexpected BaseBranch: %s", req.BaseBranch)
	}
}

// TestCreatePRResponse tests the response structure.
func TestCreatePRResponse(t *testing.T) {
	resp := &CreatePRResponse{
		Number: 1,
		State:  "open",
	}

	if resp.Number != 1 {
		t.Errorf("Expected number 1, got %d", resp.Number)
	}

	if resp.State != "open" {
		t.Errorf("Expected state 'open', got %s", resp.State)
	}
}
