// Package context provides context building for LLM prompts.
package context

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"
)

// Service implements ContextBuilder with caching support.
type Service struct {
	shortTerm *ShortTermExtractor
	longTerm  *LongTermExtractor
	ranker    *PriorityRanker
	allocator *BudgetAllocator

	// Providers (injected)
	messageProvider  MessageProvider
	episodicProvider EpisodicProvider
	prefProvider     PreferenceProvider

	// Cache (optional)
	cache CacheProvider

	// Stats
	stats *serviceStats
}

// CacheProvider provides caching functionality.
type CacheProvider interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

type serviceStats struct {
	totalBuilds  int64
	totalTokens  int64
	cacheHits    int64
	totalBuildMs int64
}

// Config configures the context builder service.
type Config struct {
	MaxTurns    int           // Max conversation turns (default: 10)
	MaxEpisodes int           // Max episodic memories (default: 3)
	MaxTokens   int           // Default max tokens (default: 4096)
	CacheTTL    time.Duration // Cache TTL (default: 5 minutes)
}

// DefaultConfig returns default configuration.
func DefaultConfig() Config {
	return Config{
		MaxTurns:    10,
		MaxEpisodes: 3,
		MaxTokens:   4096,
		CacheTTL:    5 * time.Minute,
	}
}

// NewService creates a new context builder service.
func NewService(cfg Config) *Service {
	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = 10
	}
	if cfg.MaxEpisodes <= 0 {
		cfg.MaxEpisodes = 3
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 4096
	}

	return &Service{
		shortTerm: NewShortTermExtractor(cfg.MaxTurns),
		longTerm:  NewLongTermExtractor(cfg.MaxEpisodes),
		ranker:    NewPriorityRanker(),
		allocator: NewBudgetAllocator(),
		stats:     &serviceStats{},
	}
}

// WithMessageProvider sets the message provider.
func (s *Service) WithMessageProvider(p MessageProvider) *Service {
	s.messageProvider = p
	return s
}

// WithEpisodicProvider sets the episodic memory provider.
func (s *Service) WithEpisodicProvider(p EpisodicProvider) *Service {
	s.episodicProvider = p
	return s
}

// WithPreferenceProvider sets the preference provider.
func (s *Service) WithPreferenceProvider(p PreferenceProvider) *Service {
	s.prefProvider = p
	return s
}

// WithCache sets the cache provider.
func (s *Service) WithCache(c CacheProvider) *Service {
	s.cache = c
	return s
}

// Build constructs the context for LLM inference.
func (s *Service) Build(ctx context.Context, req *ContextRequest) (*ContextResult, error) {
	start := time.Now()
	atomic.AddInt64(&s.stats.totalBuilds, 1)

	// Set defaults
	if req.MaxTokens <= 0 {
		req.MaxTokens = DefaultMaxTokens
	}

	// Allocate token budget
	hasRetrieval := len(req.RetrievalResults) > 0
	budget := s.allocator.Allocate(req.MaxTokens, hasRetrieval)

	// Build context segments
	var segments []*ContextSegment

	// 1. System prompt
	systemPrompt := s.buildSystemPrompt(req.AgentType)
	segments = append(segments, &ContextSegment{
		Content:   systemPrompt,
		Priority:  PrioritySystem,
		TokenCost: EstimateTokens(systemPrompt),
		Source:    "system",
	})

	// 2. Short-term memory (recent conversation)
	if s.messageProvider != nil && req.SessionID != "" {
		messages, err := s.shortTerm.Extract(ctx, s.messageProvider, req.SessionID)
		if err != nil {
			slog.Warn("failed to extract short-term memory", "session_id", req.SessionID, "error", err)
		}
		if len(messages) > 0 {
			recent, older := SplitByRecency(messages, 3)

			// Recent turns - high priority
			if len(recent) > 0 {
				recentText := FormatConversation(recent)
				segments = append(segments, &ContextSegment{
					Content:   recentText,
					Priority:  PriorityRecentTurns,
					TokenCost: EstimateTokens(recentText),
					Source:    "short_term",
				})
			}

			// Older turns - lower priority
			if len(older) > 0 {
				olderText := FormatConversation(older)
				segments = append(segments, &ContextSegment{
					Content:   olderText,
					Priority:  PriorityOlderTurns,
					TokenCost: EstimateTokens(olderText),
					Source:    "short_term",
				})
			}
		}
	}

	// 3. Long-term memory (episodic + preferences)
	if req.UserID > 0 {
		longTermCtx, err := s.longTerm.Extract(ctx, s.episodicProvider, s.prefProvider, req.UserID, req.CurrentQuery)
		if err != nil {
			slog.Warn("failed to extract long-term memory", "user_id", req.UserID, "error", err)
		}
		if longTermCtx != nil {
			// Episodic memories
			if len(longTermCtx.Episodes) > 0 {
				episodicText := FormatEpisodes(longTermCtx.Episodes)
				segments = append(segments, &ContextSegment{
					Content:   episodicText,
					Priority:  PriorityEpisodic,
					TokenCost: EstimateTokens(episodicText),
					Source:    "long_term",
				})
			}

			// User preferences
			if longTermCtx.Preferences != nil {
				prefsText := FormatPreferences(longTermCtx.Preferences)
				if prefsText != "" {
					segments = append(segments, &ContextSegment{
						Content:   prefsText,
						Priority:  PriorityPreferences,
						TokenCost: EstimateTokens(prefsText),
						Source:    "prefs",
					})
				}
			}
		}
	}

	// 4. Retrieval results
	if hasRetrieval {
		retrievalText := s.formatRetrieval(req.RetrievalResults)
		segments = append(segments, &ContextSegment{
			Content:   retrievalText,
			Priority:  PriorityRetrieval,
			TokenCost: EstimateTokens(retrievalText),
			Source:    "retrieval",
		})
	}

	// Prioritize and truncate to fit total budget
	// Note: System prompt is included in segments and will be prioritized highest
	finalSegments := s.ranker.RankAndTruncate(segments, budget.Total)

	// Assemble result
	result := s.assembleResult(finalSegments, budget)
	result.BuildTime = time.Since(start)

	// Update stats
	atomic.AddInt64(&s.stats.totalTokens, int64(result.TotalTokens))
	atomic.AddInt64(&s.stats.totalBuildMs, result.BuildTime.Milliseconds())

	return result, nil
}

// GetStats returns context building statistics.
func (s *Service) GetStats() *ContextStats {
	builds := atomic.LoadInt64(&s.stats.totalBuilds)
	if builds == 0 {
		return &ContextStats{}
	}

	return &ContextStats{
		TotalBuilds:      builds,
		AverageTokens:    float64(atomic.LoadInt64(&s.stats.totalTokens)) / float64(builds),
		CacheHits:        atomic.LoadInt64(&s.stats.cacheHits),
		AverageBuildTime: time.Duration(atomic.LoadInt64(&s.stats.totalBuildMs)/builds) * time.Millisecond,
	}
}

// buildSystemPrompt generates the system prompt for the agent type.
func (s *Service) buildSystemPrompt(agentType string) string {
	switch agentType {
	case "memo":
		return `你是一个智能笔记助手。帮助用户搜索、整理和管理笔记。
回答应简洁准确，优先使用检索到的笔记内容作为依据。`
	case "schedule":
		return `你是一个日程管理助手。帮助用户创建、查询和管理日程安排。
理解用户的时间表达，准确提取日期和时间信息。`
	case "amazing":
		return `你是一个综合助手。帮助用户分析问题、总结信息、提供建议。
根据上下文提供有见地的回答。`
	default:
		return `你是一个智能助手，帮助用户完成各种任务。
请根据上下文信息提供准确、有帮助的回答。`
	}
}

// formatRetrieval formats retrieval results into context.
func (s *Service) formatRetrieval(results []*RetrievalItem) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("### 相关信息\n")

	for i, item := range results {
		if i >= 5 { // Limit to top 5
			break
		}
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Content))
	}

	return sb.String()
}

// assembleResult assembles the final context result.
func (s *Service) assembleResult(segments []*ContextSegment, budget *TokenBudget) *ContextResult {
	result := &ContextResult{
		TokenBreakdown: &TokenBreakdown{},
	}

	var conversation, retrieval, prefs strings.Builder

	for _, seg := range segments {
		switch seg.Source {
		case "system":
			result.SystemPrompt = seg.Content
			result.TokenBreakdown.SystemPrompt = seg.TokenCost
		case "short_term":
			conversation.WriteString(seg.Content)
			result.TokenBreakdown.ShortTermMemory += seg.TokenCost
		case "long_term":
			conversation.WriteString(seg.Content)
			result.TokenBreakdown.LongTermMemory += seg.TokenCost
		case "retrieval":
			retrieval.WriteString(seg.Content)
			result.TokenBreakdown.Retrieval = seg.TokenCost
		case "prefs":
			prefs.WriteString(seg.Content)
			result.TokenBreakdown.UserPrefs = seg.TokenCost
		}

		result.TotalTokens += seg.TokenCost
	}

	result.ConversationContext = conversation.String()
	result.RetrievalContext = retrieval.String()
	result.UserPreferences = prefs.String()

	return result
}

// Ensure Service implements ContextBuilder.
var _ ContextBuilder = (*Service)(nil)
