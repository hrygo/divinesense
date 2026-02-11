package runner

import (
	"sync"
	"time"
)

// SessionStats collects session-level statistics for Geek/Evolution modes.
// SessionStats 收集极客/进化模式的会话级别统计数据。
type SessionStats struct {
	mu                   sync.Mutex
	SessionID            string
	StartTime            time.Time
	TotalDurationMs      int64
	ThinkingDurationMs   int64
	ToolDurationMs       int64
	GenerationDurationMs int64
	InputTokens          int32
	OutputTokens         int32
	CacheWriteTokens     int32
	CacheReadTokens      int32
	ToolCallCount        int32
	ToolsUsed            map[string]bool
	FilesModified        int32
	FilePaths            []string
	filePathsSet         map[string]bool // O(1) deduplication for file paths

	// Current tool tracking
	currentToolStart time.Time
	currentToolName  string
	currentToolID    string

	// Phase tracking for duration breakdown
	thinkingStart   time.Time
	generationStart time.Time
	hasGeneration   bool // Tracks if any content was generated
}

// RecordToolUse records the start of a tool call.
func (s *SessionStats) RecordToolUse(toolName, toolID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentToolStart = time.Now()
	s.currentToolName = toolName
	s.currentToolID = toolID
	// Ensure ToolsUsed map is initialized (concurrency safety)
	if s.ToolsUsed == nil {
		s.ToolsUsed = make(map[string]bool)
	}
}

// RecordToolResult records the end of a tool call.
func (s *SessionStats) RecordToolResult() (durationMs int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.currentToolStart.IsZero() {
		duration := time.Since(s.currentToolStart)
		durationMs = duration.Milliseconds()
		s.ToolDurationMs += durationMs
		s.ToolCallCount++
		if s.currentToolName != "" {
			if s.ToolsUsed == nil {
				s.ToolsUsed = make(map[string]bool)
			}
			s.ToolsUsed[s.currentToolName] = true
		}
		s.currentToolStart = time.Time{}
		s.currentToolName = ""
		s.currentToolID = ""
	}
	return durationMs
}

// RecordTokens records token usage.
func (s *SessionStats) RecordTokens(input, output, cacheWrite, cacheRead int32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.InputTokens += input
	s.OutputTokens += output
	s.CacheWriteTokens += cacheWrite
	s.CacheReadTokens += cacheRead
}

// StartThinking marks the start of the thinking phase.
// StartThinking 标记思考阶段的开始。
func (s *SessionStats) StartThinking() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.thinkingStart.IsZero() {
		s.thinkingStart = time.Now()
	}
}

// EndThinking marks the end of the thinking phase and records its duration.
// EndThinking 标记思考阶段的结束并记录其持续时间。
func (s *SessionStats) EndThinking() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.thinkingStart.IsZero() {
		s.ThinkingDurationMs += time.Since(s.thinkingStart).Milliseconds()
		s.thinkingStart = time.Time{} // Reset for next thinking phase
	}
}

// StartGeneration marks the start of the generation phase.
// StartGeneration 标记生成阶段的开始。
func (s *SessionStats) StartGeneration() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.generationStart.IsZero() {
		s.generationStart = time.Now()
		s.hasGeneration = true
	}
}

// EndGeneration marks the end of the generation phase and records its duration.
// EndGeneration 标记生成阶段的结束并记录其持续时间。
func (s *SessionStats) EndGeneration() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.generationStart.IsZero() {
		s.GenerationDurationMs += time.Since(s.generationStart).Milliseconds()
		s.generationStart = time.Time{} // Reset for next generation phase
	}
}

// RecordFileModification records that a file was modified.
// Uses O(1) map lookup for deduplication instead of O(n) linear scan.
func (s *SessionStats) RecordFileModification(filePath string) {
	if filePath == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Initialize map if needed
	if s.filePathsSet == nil {
		s.filePathsSet = make(map[string]bool)
	}

	// O(1) deduplication check
	if s.filePathsSet[filePath] {
		return // Already recorded
	}

	s.filePathsSet[filePath] = true
	s.FilePaths = append(s.FilePaths, filePath)
	s.FilesModified++
}

// ToSummary converts stats to a summary map for JSON serialization.
func (s *SessionStats) ToSummary() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	tools := make([]string, 0, len(s.ToolsUsed))
	for tool := range s.ToolsUsed {
		tools = append(tools, tool)
	}

	return map[string]any{
		"session_id":               s.SessionID,
		"total_duration_ms":        s.TotalDurationMs,
		"thinking_duration_ms":     s.ThinkingDurationMs,
		"tool_duration_ms":         s.ToolDurationMs,
		"generation_duration_ms":   s.GenerationDurationMs,
		"total_input_tokens":       s.InputTokens,
		"total_output_tokens":      s.OutputTokens,
		"total_cache_write_tokens": s.CacheWriteTokens,
		"total_cache_read_tokens":  s.CacheReadTokens,
		"tool_call_count":          s.ToolCallCount,
		"tools_used":               tools,
		"files_modified":           s.FilesModified,
		"file_paths":               s.FilePaths,
		"status":                   "success",
	}
}

// FinalizeDuration finalizes any ongoing phase tracking and returns the final stats.
// FinalizeDuration 完成任何正在进行的阶段追踪并返回最终统计数据。
func (s *SessionStats) FinalizeDuration() *SessionStats {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalThinking := s.ThinkingDurationMs
	totalGeneration := s.GenerationDurationMs

	if !s.thinkingStart.IsZero() {
		totalThinking += time.Since(s.thinkingStart).Milliseconds()
	}
	if !s.generationStart.IsZero() {
		totalGeneration += time.Since(s.generationStart).Milliseconds()
	}

	return &SessionStats{
		SessionID:            s.SessionID,
		StartTime:            s.StartTime,
		TotalDurationMs:      s.TotalDurationMs,
		ThinkingDurationMs:   totalThinking,
		ToolDurationMs:       s.ToolDurationMs,
		GenerationDurationMs: totalGeneration,
		InputTokens:          s.InputTokens,
		OutputTokens:         s.OutputTokens,
		CacheWriteTokens:     s.CacheWriteTokens,
		CacheReadTokens:      s.CacheReadTokens,
		ToolCallCount:        s.ToolCallCount,
		ToolsUsed:            s.ToolsUsed,
		FilesModified:        s.FilesModified,
		FilePaths:            s.FilePaths,
	}
}
