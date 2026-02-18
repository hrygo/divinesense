package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hrygo/divinesense/ai/core/retrieval"
	"github.com/hrygo/divinesense/ai/timeout"
	"github.com/hrygo/divinesense/server/queryengine"
)

const (
	// Default search limit for memo search results.
	defaultSearchLimit = 10

	// Maximum search limit to prevent excessive results.
	maxSearchLimit = 50

	// Default minimum relevance score for search results.
	defaultMinScore = 0.5
)

// JSON field name mappings for camelCase to snake_case compatibility.
// Some LLMs generate camelCase (minScore) while we expect snake_case (min_score).
var memoFieldNameMappings = map[string]string{
	"minScore": "min_score",
}

// normalizeMemoJSONFields converts camelCase keys to snake_case for LLM compatibility.
func normalizeMemoJSONFields(inputJSON string) string {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(inputJSON), &raw); err != nil {
		return inputJSON
	}

	normalized := make(map[string]interface{})
	for key, value := range raw {
		newKey := key
		if mapped, ok := memoFieldNameMappings[key]; ok {
			newKey = mapped
		}
		normalized[newKey] = value
	}

	result, err := json.Marshal(normalized)
	if err != nil {
		return inputJSON
	}
	return string(result)
}

// MemoSearchTool searches for memos using semantic and keyword search.
// MemoSearchTool 使用语义和关键词搜索来查找笔记。
type MemoSearchTool struct {
	retriever    *retrieval.AdaptiveRetriever
	userIDGetter func(ctx context.Context) int32
	classifier   *MemoQueryClassifier
}

// NewMemoSearchTool creates a new memo search tool.
// NewMemoSearchTool 创建一个新的笔记搜索工具。
func NewMemoSearchTool(
	retriever *retrieval.AdaptiveRetriever,
	userIDGetter func(ctx context.Context) int32,
) (*MemoSearchTool, error) {
	if retriever == nil {
		return nil, fmt.Errorf("retriever cannot be nil")
	}
	if userIDGetter == nil {
		return nil, fmt.Errorf("userIDGetter cannot be nil")
	}

	return &MemoSearchTool{
		retriever:    retriever,
		userIDGetter: userIDGetter,
		classifier:   NewMemoQueryClassifier(),
	}, nil
}

// Name returns the name of the tool.
// Name 返回工具名称。
func (t *MemoSearchTool) Name() string {
	return "memo_search"
}

// Description returns a description of what the tool does.
// Description 返回工具描述。
func (t *MemoSearchTool) Description() string {
	return `Search memos using intelligent hybrid retrieval strategy.

This tool automatically selects the best retrieval strategy based on query intent:
- List/browsing queries → Time-ordered list
- Filter queries → Time/tag-based filtering
- Keyword queries → BM25 text search
- Semantic queries → Vector similarity search
- Complex queries → Hybrid retrieval + Rerank

Input: {"query": "keywords", "limit": 10}
- query (required): search keywords in Chinese or English
- limit (optional): max results, default 10

Output: Ranked list of memos with content and UID.`
}

// MemoSearchInput represents the input for memo search.
// MemoSearchInput 表示笔记搜索的输入。
type MemoSearchInput struct {
	Query    string  `json:"query"`
	Strategy string  `json:"strategy,omitempty"`
	Limit    int     `json:"limit,omitempty"`
	MinScore float32 `json:"min_score,omitempty"`
}

// Run executes the memo search tool.
// Run 执行笔记搜索工具。
func (t *MemoSearchTool) Run(ctx context.Context, input string) (string, error) {
	// Add timeout protection for search operation
	ctx, cancel := context.WithTimeout(ctx, timeout.ToolExecutionTimeout)
	defer cancel()

	// Normalize JSON field names (camelCase -> snake_case) for LLM compatibility
	normalizedInput := normalizeMemoJSONFields(input)

	// Parse input
	var searchInput MemoSearchInput
	if err := json.Unmarshal([]byte(normalizedInput), &searchInput); err != nil {
		return "", fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate query - empty query is treated as "search all"
	if strings.TrimSpace(searchInput.Query) == "" {
		searchInput.Query = "*" // Search all memos when query is empty
	}

	// Set defaults
	if searchInput.Limit <= 0 {
		searchInput.Limit = defaultSearchLimit
	}
	if searchInput.Limit > maxSearchLimit {
		searchInput.Limit = maxSearchLimit
	}
	if searchInput.MinScore <= 0 {
		searchInput.MinScore = defaultMinScore
	}

	// Get user ID
	userID := t.userIDGetter(ctx)

	// Smart strategy selection: classify query intent and route accordingly
	// 智能策略选择：分类查询意图并路由到对应的检索策略
	strategy := searchInput.Strategy
	if strategy == "" {
		intent := t.classifier.Classify(searchInput.Query)
		strategy = intent.ToStrategy()
	}

	// Extract tags from query for tag filtering
	tags := t.classifier.ExtractTags(searchInput.Query)

	// Execute search
	opts := &retrieval.RetrievalOptions{
		Query:           searchInput.Query,
		UserID:          userID,
		Strategy:        strategy,
		Limit:           searchInput.Limit,
		MinScore:        searchInput.MinScore,
		Tags:            tags,
		ExcludeComments: true, // Exclude comments from search results
	}

	// Add time range and tags for filter queries
	if strategy == "memo_filter_only" {
		start, end, ok := t.classifier.ExtractTimeFilter(searchInput.Query)
		if ok {
			opts.TimeRange = &queryengine.TimeRange{
				Start: start,
				End:   end,
			}
		}
	}

	results, err := t.retriever.Retrieve(ctx, opts)
	if err != nil {
		// Check for timeout specifically
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("search timeout after %v", timeout.ToolExecutionTimeout)
		}
		return "", fmt.Errorf("search failed: %w", err)
	}

	// Filter only memo results (exclude schedules)
	var memoResults []*retrieval.SearchResult
	for _, result := range results {
		if result.Type == "memo" {
			memoResults = append(memoResults, result)
		}
	}

	// Format results
	if len(memoResults) == 0 {
		return fmt.Sprintf("No memos found matching query: %s", searchInput.Query), nil
	}

	// Build response
	var response strings.Builder
	fmt.Fprintf(&response, "Found %d memo(s) matching query: %s\n\n", len(memoResults), searchInput.Query)

	for i, result := range memoResults {
		fmt.Fprintf(&response, "%d. %s\n", i+1, result.Content)

		// Add memo UID if available
		if result.Memo != nil && result.Memo.UID != "" {
			fmt.Fprintf(&response, "   UID: %s\n", result.Memo.UID)
		}

		fmt.Fprintf(&response, "\n")
	}

	return response.String(), nil
}

// MemoSummary represents a simplified memo for query results.
type MemoSummary struct {
	UID     string  `json:"uid"`
	Content string  `json:"content"`
	Score   float32 `json:"score"`
}

// MemoSearchToolResult represents the structured result of memo search.
// MemoSearchToolResult 表示笔记搜索的结构化结果。
type MemoSearchToolResult struct {
	Query   string        `json:"query"`
	Memos   []MemoSummary `json:"memos"`
	Count   int           `json:"count"`
	Quality string        `json:"quality,omitempty"` // "high", "medium", "low" - for frontend display decision
}

// RunWithStructuredResult executes the tool and returns a structured result.
// RunWithStructuredResult 执行工具并返回结构化结果。
func (t *MemoSearchTool) RunWithStructuredResult(ctx context.Context, input string) (*MemoSearchToolResult, error) {
	// Add timeout protection for search operation
	ctx, cancel := context.WithTimeout(ctx, timeout.ToolExecutionTimeout)
	defer cancel()

	// Normalize JSON field names (camelCase -> snake_case) for LLM compatibility
	normalizedInput := normalizeMemoJSONFields(input)

	// Parse input
	var searchInput MemoSearchInput
	if err := json.Unmarshal([]byte(normalizedInput), &searchInput); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}

	// Validate query - empty query is treated as "search all"
	if strings.TrimSpace(searchInput.Query) == "" {
		searchInput.Query = "*" // Search all memos when query is empty
	}

	// Set defaults using defined constants
	if searchInput.Limit <= 0 {
		searchInput.Limit = defaultSearchLimit
	}
	if searchInput.Limit > maxSearchLimit {
		searchInput.Limit = maxSearchLimit
	}
	if searchInput.MinScore <= 0 {
		searchInput.MinScore = defaultMinScore
	}

	// Get user ID
	userID := t.userIDGetter(ctx)

	// Set strategy
	strategy := searchInput.Strategy
	if strategy == "" {
		strategy = "memo_semantic_only"
	}

	// Execute search
	opts := &retrieval.RetrievalOptions{
		Query:           searchInput.Query,
		UserID:          userID,
		Strategy:        strategy,
		Limit:           searchInput.Limit,
		MinScore:        searchInput.MinScore,
		ExcludeComments: true, // Exclude comments from search results
	}

	results, err := t.retriever.Retrieve(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Filter only memo results and convert to summaries
	var memos []MemoSummary
	for _, result := range results {
		if result.Type == "memo" && result.Memo != nil {
			memos = append(memos, MemoSummary{
				UID:     result.Memo.UID,
				Content: result.Content,
				Score:   result.Score,
			})
		}
	}

	// Determine quality level based on top score for frontend display decision
	quality := "high"
	if len(memos) > 0 && memos[0].Score < 0.70 {
		quality = "low"
	} else if len(memos) > 0 && memos[0].Score < 0.90 {
		quality = "medium"
	}

	return &MemoSearchToolResult{
		Query:   searchInput.Query,
		Memos:   memos,
		Count:   len(memos),
		Quality: quality,
	}, nil
}
