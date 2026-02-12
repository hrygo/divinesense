package retrieval

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/server/queryengine"
	"github.com/hrygo/divinesense/store"
)

// RRF constants.
const (
	// A value of 60 is commonly used in information retrieval.
	RRFK = 60
)

// 根据查询复杂度和结果质量动态调整检索策略.
type AdaptiveRetriever struct {
	store            *store.Store
	embeddingService ai.EmbeddingService
	rerankerService  ai.RerankerService
}

// SearchResult 检索结果.
type SearchResult struct {
	Memo     *store.Memo
	Schedule *store.Schedule
	Type     string
	Content  string
	ID       int64
	Score    float32
}

// RetrievalOptions 检索选项.
type RetrievalOptions struct {
	TimeRange         *queryengine.TimeRange
	Logger            *slog.Logger
	Query             string
	Strategy          string
	RequestID         string
	Limit             int
	UserID            int32
	MinScore          float32
	ScheduleQueryMode queryengine.ScheduleQueryMode
}

// NewAdaptiveRetriever 创建自适应检索器.
func NewAdaptiveRetriever(
	st *store.Store,
	embeddingService ai.EmbeddingService,
	rerankerService ai.RerankerService,
) *AdaptiveRetriever {
	return &AdaptiveRetriever{
		store:            st,
		embeddingService: embeddingService,
		rerankerService:  rerankerService,
	}
}

// Retrieve 自适应检索主入口.
func (r *AdaptiveRetriever) Retrieve(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	if opts == nil {
		opts = &RetrievalOptions{
			Strategy: "hybrid_standard",
			Limit:    10,
			MinScore: 0.5,
		}
	}

	// 输入验证：P0 改进 - 添加查询长度限制
	if len(opts.Query) > 1000 {
		return nil, fmt.Errorf("query too long: %d characters (max 1000)", len(opts.Query))
	}

	// 初始化日志记录器
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.RequestID == "" {
		opts.RequestID = generateRequestID()
	}

	// 根据路由策略选择检索路径
	switch opts.Strategy {
	case "schedule_bm25_only":
		return r.scheduleBM25Only(ctx, opts)

	case "memo_list_only":
		return r.memoListOnly(ctx, opts)

	case "memo_filter_only":
		return r.memoFilterOnly(ctx, opts)

	case "memo_bm25_only":
		return r.memoBM25Only(ctx, opts)

	case "memo_semantic_only":
		return r.memoSemanticOnly(ctx, opts)

	case "hybrid_bm25_weighted":
		return r.hybridBM25Weighted(ctx, opts)

	case "hybrid_with_time_filter":
		return r.hybridWithTimeFilter(ctx, opts)

	case "hybrid_standard":
		return r.hybridStandard(ctx, opts)

	case "full_pipeline_with_reranker":
		return r.fullPipelineWithReranker(ctx, opts)

	default:
		// 默认使用标准混合检索
		return r.hybridStandard(ctx, opts)
	}
}

// scheduleBM25Only 纯日程查询（BM25 + 时间过滤）.
func (r *AdaptiveRetriever) scheduleBM25Only(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "schedule_bm25_only",
		"user_id", opts.UserID,
	)

	// 构建查询条件
	findSchedule := &store.FindSchedule{
		CreatorID: &opts.UserID,
	}

	// P1: 设置查询模式（将 queryengine.ScheduleQueryMode 转换为 int32）
	if opts.ScheduleQueryMode != queryengine.AutoQueryMode {
		mode := int32(opts.ScheduleQueryMode)
		findSchedule.QueryMode = &mode
	}

	// 添加时间过滤（P0 改进：添加 nil 检查和验证）
	if opts.TimeRange != nil {
		// 验证时间范围
		if !opts.TimeRange.ValidateTimeRange() {
			opts.Logger.WarnContext(ctx, "Invalid time range",
				"request_id", opts.RequestID,
				"start", opts.TimeRange.Start,
				"end", opts.TimeRange.End,
			)
			return nil, fmt.Errorf("invalid time range: start=%v, end=%v", opts.TimeRange.Start, opts.TimeRange.End)
		}

		startTs := opts.TimeRange.Start.Unix()
		endTs := opts.TimeRange.End.Unix()
		findSchedule.StartTs = &startTs
		findSchedule.EndTs = &endTs
	}

	// 查询日程
	schedules, err := r.store.ListSchedules(ctx, findSchedule)
	if err != nil {
		opts.Logger.ErrorContext(ctx, "Failed to list schedules",
			"request_id", opts.RequestID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	// P1 改进：内存优化 - 预分配切片容量
	results := make([]*SearchResult, 0, len(schedules))
	for _, schedule := range schedules {
		results = append(results, &SearchResult{
			ID:       int64(schedule.ID),
			Type:     "schedule",
			Score:    1.0, // 日程查询默认高分
			Content:  schedule.Title,
			Schedule: schedule,
		})
	}

	// P1 改进：内存优化 - 释放不再需要的大对象引用
	// 如果 Schedule 描述很大，可以只保留必要的字段
	for _, result := range results {
		if result.Schedule != nil && len(result.Schedule.Description) > 10000 {
			// 描述超过 10KB，截断以减少内存占用
			result.Content = result.Schedule.Title
			result.Schedule = nil // 释放完整 Schedule 对象
		}
	}

	opts.Logger.InfoContext(ctx, "Schedule retrieval completed",
		"request_id", opts.RequestID,
		"result_count", len(results),
	)

	return results, nil
}

// memoListOnly 纯 SQL 列表查询（最快，用于"列出所有笔记"类查询）.
// 无需向量搜索或 BM25，直接从数据库列表返回.
func (r *AdaptiveRetriever) memoListOnly(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "memo_list_only",
		"user_id", opts.UserID,
	)

	// 构建 SQL 查询条件
	findMemo := &store.FindMemo{
		CreatorID:        &opts.UserID,
		OrderByUpdatedTs: true,
	}

	// 设置限制
	limit := opts.Limit
	if limit <= 0 {
		limit = 20 // 默认返回 20 条
	}
	findMemo.Limit = &limit

	// 执行 SQL 查询
	memos, err := r.store.ListMemos(ctx, findMemo)
	if err != nil {
		opts.Logger.ErrorContext(ctx, "Failed to list memos",
			"request_id", opts.RequestID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list memos: %w", err)
	}

	// 转换为 SearchResult
	results := make([]*SearchResult, 0, len(memos))
	for _, memo := range memos {
		results = append(results, &SearchResult{
			ID:      int64(memo.ID),
			Type:    "memo",
			Score:   1.0, // 列表查询默认满分
			Content: memo.Content,
			Memo:    memo,
		})
	}

	opts.Logger.InfoContext(ctx, "Memo list retrieval completed",
		"request_id", opts.RequestID,
		"result_count", len(results),
	)

	return results, nil
}

// memoFilterOnly SQL 过滤查询（按时间、标签等过滤）.
// 用于"今天的笔记"、"本周笔记"等过滤类查询.
// 使用 CEL 表达式过滤 + SQL 排序.
func (r *AdaptiveRetriever) memoFilterOnly(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "memo_filter_only",
		"user_id", opts.UserID,
	)

	// 构建 SQL 查询条件
	findMemo := &store.FindMemo{
		CreatorID:        &opts.UserID,
		OrderByUpdatedTs: true,
	}

	// 添加时间过滤（使用 CEL 表达式）
	if opts.TimeRange != nil {
		if opts.TimeRange.Start.Unix() > 0 && opts.TimeRange.End.Unix() > 0 {
			// 使用 CEL 表达式过滤时间范围
			// created_ts >= start AND created_ts <= end
			startTs := opts.TimeRange.Start.Unix()
			endTs := opts.TimeRange.End.Unix()
			filterExpr := fmt.Sprintf("created_ts >= %d && created_ts <= %d", startTs, endTs)
			findMemo.Filters = []string{filterExpr}
		}
	}

	// 设置限制
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	findMemo.Limit = &limit

	// 执行 SQL 查询
	memos, err := r.store.ListMemos(ctx, findMemo)
	if err != nil {
		opts.Logger.ErrorContext(ctx, "Failed to filter memos",
			"request_id", opts.RequestID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to filter memos: %w", err)
	}

	// 转换为 SearchResult
	results := make([]*SearchResult, 0, len(memos))
	for _, memo := range memos {
		results = append(results, &SearchResult{
			ID:      int64(memo.ID),
			Type:    "memo",
			Score:   1.0,
			Content: memo.Content,
			Memo:    memo,
		})
	}

	opts.Logger.InfoContext(ctx, "Memo filter retrieval completed",
		"request_id", opts.RequestID,
		"result_count", len(results),
	)

	return results, nil
}

// memoBM25Only 纯 BM25 关键词搜索（无需向量检索）.
// 用于短关键词、技术术语等精确匹配场景.
func (r *AdaptiveRetriever) memoBM25Only(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "memo_bm25_only",
		"user_id", opts.UserID,
	)

	// 设置 BM25 搜索参数
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	// 执行 BM25 搜索
	// Note: MinScore=0 means no filtering - rely on DB's ORDER BY score DESC + LIMIT
	// This is best practice since DB already returns most relevant results
	bm25Results, err := r.store.BM25Search(ctx, &store.BM25SearchOptions{
		UserID: opts.UserID,
		Query:  opts.Query,
		Limit:  limit,
		// MinScore: 0 (default) - no post-filter needed
	})
	if err != nil {
		opts.Logger.ErrorContext(ctx, "BM25 search failed",
			"request_id", opts.RequestID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// 转换为 SearchResult
	results := make([]*SearchResult, 0, len(bm25Results))
	for _, bm25Result := range bm25Results {
		results = append(results, &SearchResult{
			ID:      int64(bm25Result.Memo.ID),
			Type:    "memo",
			Score:   bm25Result.Score,
			Content: bm25Result.Memo.Content,
			Memo:    bm25Result.Memo,
		})
	}

	opts.Logger.InfoContext(ctx, "BM25 retrieval completed",
		"request_id", opts.RequestID,
		"result_count", len(results),
	)

	return results, nil
}

// memoSemanticOnly 纯笔记查询（语义向量）.
func (r *AdaptiveRetriever) memoSemanticOnly(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "memo_semantic_only",
		"user_id", opts.UserID,
	)

	// 生成查询向量
	queryVector, err := r.embeddingService.Embed(ctx, opts.Query)
	if err != nil {
		opts.Logger.ErrorContext(ctx, "Failed to embed query",
			"request_id", opts.RequestID,
			"error", err,
			"query_length", len(opts.Query),
		)
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// 第一阶段：快速检索 Top 5
	limit := 5
	if opts.Limit > 0 {
		limit = opts.Limit
	}

	// Optimize: Only search memos from the last 90 days to reduce candidates
	// This significantly improves performance for large datasets
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90).Unix()

	vectorResults, err := r.store.VectorSearch(ctx, &store.VectorSearchOptions{
		UserID:       opts.UserID,
		Vector:       queryVector,
		Limit:        limit,
		CreatedAfter: ninetyDaysAgo, // Only search recent memos
	})
	if err != nil {
		opts.Logger.ErrorContext(ctx, "Vector search failed",
			"request_id", opts.RequestID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// 评估结果质量
	results := r.convertVectorResults(vectorResults)

	quality := r.evaluateQuality(results)
	opts.Logger.InfoContext(ctx, "Evaluated result quality",
		"request_id", opts.RequestID,
		"quality_level", quality.String(),
		"result_count", len(results),
	)

	// 根据质量决定是否扩展
	if quality == MediumQuality && opts.Limit > 5 {
		// 扩展到 Top 20 (with same time filter for consistency)
		ninetyDaysAgo := time.Now().AddDate(0, 0, -90).Unix()
		moreResults, err := r.store.VectorSearch(ctx, &store.VectorSearchOptions{
			UserID:       opts.UserID,
			Vector:       queryVector,
			Limit:        20,
			CreatedAfter: ninetyDaysAgo,
		})
		if err == nil {
			// 合并结果
			results = r.mergeResults(results, r.convertVectorResults(moreResults), opts.Limit)
			opts.Logger.DebugContext(ctx, "Expanded results",
				"request_id", opts.RequestID,
				"new_count", len(results),
			)
		}
	}

	// 过滤低分结果
	filtered := r.filterByScore(results, opts.MinScore)
	opts.Logger.InfoContext(ctx, "Semantic retrieval completed",
		"request_id", opts.RequestID,
		"final_count", len(filtered),
		"min_score", opts.MinScore,
	)

	return filtered, nil
}

// hybridBM25Weighted 混合检索（BM25 加权）.
func (r *AdaptiveRetriever) hybridBM25Weighted(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "hybrid_bm25_weighted",
		"user_id", opts.UserID,
	)

	// BM25 权重更高（0.7），语义权重更低（0.3）
	return r.hybridSearch(ctx, opts, 0.3)
}

// hybridWithTimeFilter 混合检索（时间过滤）.
func (r *AdaptiveRetriever) hybridWithTimeFilter(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "hybrid_with_time_filter",
		"user_id", opts.UserID,
	)

	// 标准混合检索 + 时间过滤
	results, err := r.hybridSearch(ctx, opts, 0.5)
	if err != nil {
		return nil, err
	}

	// 如果指定了时间范围，过滤日程结果（P0 改进：添加 nil 检查）
	if opts.TimeRange != nil {
		// P1 改进：内存优化 - 预分配容量
		filtered := make([]*SearchResult, 0, len(results))
		for _, result := range results {
			if result.Type == "memo" {
				filtered = append(filtered, result)
			} else if result.Type == "schedule" && result.Schedule != nil {
				scheduleTime := time.Unix(result.Schedule.StartTs, 0)
				if opts.TimeRange.Contains(scheduleTime) {
					filtered = append(filtered, result)
				}
			}
		}
		// P1 改进：内存优化 - 用新切片替换旧切片，让旧切片可被 GC
		results = filtered
	}

	return results, nil
}

// hybridStandard 标准混合检索（BM25 + 语义）.
func (r *AdaptiveRetriever) hybridStandard(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "hybrid_standard",
		"user_id", opts.UserID,
	)

	// BM25 和语义权重相等（0.5 + 0.5）
	return r.hybridSearch(ctx, opts, 0.5)
}

// fullPipelineWithReranker 完整流程（混合检索 + Reranker）.
func (r *AdaptiveRetriever) fullPipelineWithReranker(ctx context.Context, opts *RetrievalOptions) ([]*SearchResult, error) {
	opts.Logger.InfoContext(ctx, "Using retrieval strategy",
		"request_id", opts.RequestID,
		"strategy", "full_pipeline_with_reranker",
		"user_id", opts.UserID,
	)

	// 第一步：混合检索 Top 20
	hybridResults, err := r.hybridSearch(ctx, opts, 0.5)
	if err != nil {
		return nil, err
	}

	// 第二步：检查是否需要重排
	if !r.shouldRerank(opts.Query, hybridResults) {
		opts.Logger.InfoContext(ctx, "Skipping reranker (not needed)",
			"request_id", opts.RequestID,
			"reason", "simple_query_or_few_results",
		)
		// 不需要重排，直接返回 Top K
		return r.truncateResults(hybridResults, opts.Limit), nil
	}

	// 第三步：Reranker 重排序
	opts.Logger.InfoContext(ctx, "Applying reranker",
		"request_id", opts.RequestID,
		"result_count", len(hybridResults),
	)

	// 准备文档
	// P1 改进：内存优化 - 预分配容量
	documents := make([]string, 0, len(hybridResults))
	for _, result := range hybridResults {
		// P1 改进：内存优化 - 限制文档长度
		content := result.Content
		if len(content) > 5000 {
			// 内容超过 5000 字符，截断以减少内存和 API 成本
			content = content[:5000]
		}
		documents = append(documents, content)
	}

	// 调用 Reranker
	rerankResults, err := r.rerankerService.Rerank(ctx, opts.Query, documents, opts.Limit)
	if err != nil {
		opts.Logger.WarnContext(ctx, "Reranker failed, using hybrid results",
			"request_id", opts.RequestID,
			"error", err,
		)
		// 降级：返回原始结果
		return r.truncateResults(hybridResults, opts.Limit), nil
	}

	// 重新排序
	// P1 改进：内存优化 - 预分配容量
	reordered := make([]*SearchResult, 0, len(rerankResults))
	for _, rr := range rerankResults {
		if rr.Index < len(hybridResults) {
			result := hybridResults[rr.Index]
			result.Score = rr.Score // 更新分数
			reordered = append(reordered, result)
		}
	}

	opts.Logger.InfoContext(ctx, "Reranker completed",
		"request_id", opts.RequestID,
		"result_count", len(reordered),
	)

	// P1 改进：内存优化 - 释放不需要的大对象
	// 清空 documents 以便 GC 回收
	for i := range documents {
		documents[i] = ""
	}

	return reordered, nil
}

// 使用 RRF (Reciprocal Rank Fusion) 融合 BM25 和向量检索结果.
func (r *AdaptiveRetriever) hybridSearch(ctx context.Context, opts *RetrievalOptions, semanticWeight float32) ([]*SearchResult, error) {
	// 并行执行 BM25 和向量检索
	type vectorResult struct {
		err     error
		results []*store.MemoWithScore
	}
	type bm25Result struct {
		err     error
		results []*store.BM25Result
	}

	vectorCh := make(chan vectorResult, 1)
	bm25Ch := make(chan bm25Result, 1)

	// 并行执行向量检索
	go func() {
		queryVector, err := r.embeddingService.Embed(ctx, opts.Query)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case vectorCh <- vectorResult{
				err:     fmt.Errorf("failed to embed query: %w", err),
				results: nil,
			}:
			}
			return
		}

		// Add time filter for optimized performance
		ninetyDaysAgo := time.Now().AddDate(0, 0, -90).Unix()
		results, err := r.store.VectorSearch(ctx, &store.VectorSearchOptions{
			UserID:       opts.UserID,
			Vector:       queryVector,
			Limit:        20,
			CreatedAfter: ninetyDaysAgo,
		})
		select {
		case <-ctx.Done():
		case vectorCh <- vectorResult{
			err:     err,
			results: results,
		}:
		}
	}()

	// 并行执行 BM25 检索
	go func() {
		results, err := r.store.BM25Search(ctx, &store.BM25SearchOptions{
			UserID:   opts.UserID,
			Query:    opts.Query,
			Limit:    20,
			MinScore: 0.1,
		})
		select {
		case <-ctx.Done():
		case bm25Ch <- bm25Result{
			err:     err,
			results: results,
		}:
		}
	}()

	// 等待两个检索完成
	vectorRes := <-vectorCh
	bm25Res := <-bm25Ch

	// 处理错误
	if vectorRes.err != nil && bm25Res.err != nil {
		return nil, fmt.Errorf("both vector and BM25 search failed: vector=%w, bm25=%w", vectorRes.err, bm25Res.err)
	}

	// 如果其中一个失败，使用另一个的结果
	if vectorRes.err != nil {
		opts.Logger.WarnContext(ctx, "Vector search failed, using BM25 only",
			"request_id", opts.RequestID,
			"error", vectorRes.err,
		)
		return r.convertBM25Results(bm25Res.results), nil //nolint:nilerr // Intentional fallback
	}
	if bm25Res.err != nil {
		opts.Logger.WarnContext(ctx, "BM25 search failed, using vector only",
			"request_id", opts.RequestID,
			"error", bm25Res.err,
		)
		return r.convertVectorResults(vectorRes.results), nil //nolint:nilerr // Intentional fallback
	}

	// 使用 RRF 融合两个结果列表
	results := r.rrfFusion(vectorRes.results, bm25Res.results, semanticWeight)

	return results, nil
}

// convertVectorResults 转换向量检索结果.
func (r *AdaptiveRetriever) convertVectorResults(results []*store.MemoWithScore) []*SearchResult {
	searchResults := make([]*SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = &SearchResult{
			ID:      int64(r.Memo.ID),
			Type:    "memo",
			Score:   r.Score,
			Content: r.Memo.Content,
			Memo:    r.Memo,
		}
	}
	return searchResults
}

// convertBM25Results 转换 BM25 检索结果.
func (r *AdaptiveRetriever) convertBM25Results(results []*store.BM25Result) []*SearchResult {
	searchResults := make([]*SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = &SearchResult{
			ID:      int64(r.Memo.ID),
			Type:    "memo",
			Score:   r.Score,
			Content: r.Memo.Content,
			Memo:    r.Memo,
		}
	}
	return searchResults
}

// 其中 k 是常数 (通常取 60)，rank_i(d) 是文档在第 i 个列表中的排名.
func (r *AdaptiveRetriever) rrfFusion(vectorResults []*store.MemoWithScore, bm25Results []*store.BM25Result, semanticWeight float32) []*SearchResult {
	// 用于存储每个文档的 RRF 分数
	type rrfScore struct {
		memo       *store.Memo
		id         int64
		vectorRank int
		bm25Rank   int
		score      float32
	}

	scores := make(map[int64]*rrfScore)

	// 处理向量检索结果 (排名从 1 开始)
	for i, v := range vectorResults {
		rank := i + 1
		if existing, ok := scores[int64(v.Memo.ID)]; ok {
			existing.vectorRank = rank
		} else {
			scores[int64(v.Memo.ID)] = &rrfScore{
				id:         int64(v.Memo.ID),
				memo:       v.Memo,
				vectorRank: rank,
				bm25Rank:   -1, // 不在 BM25 结果中
			}
		}
	}

	// 处理 BM25 检索结果
	for i, b := range bm25Results {
		rank := i + 1
		if existing, ok := scores[int64(b.Memo.ID)]; ok {
			existing.bm25Rank = rank
		} else {
			scores[int64(b.Memo.ID)] = &rrfScore{
				id:         int64(b.Memo.ID),
				memo:       b.Memo,
				vectorRank: -1, // 不在向量结果中
				bm25Rank:   rank,
			}
		}
	}

	// 计算 RRF 分数
	// semanticWeight 控制 BM25 和向量检索的权重平衡
	// semanticWeight = 0.5 表示两者权重相等
	bm25Weight := 1.0 - semanticWeight

	for _, s := range scores {
		// 向量检索贡献
		if s.vectorRank > 0 {
			s.score += semanticWeight / (float32(RRFK) + float32(s.vectorRank))
		}
		// BM25 检索贡献
		if s.bm25Rank > 0 {
			s.score += bm25Weight / (float32(RRFK) + float32(s.bm25Rank))
		}
	}

	// 转换为 SearchResult 列表并按 RRF 分数排序
	results := make([]*SearchResult, 0, len(scores))
	for _, s := range scores {
		results = append(results, &SearchResult{
			ID:      s.id,
			Type:    "memo",
			Score:   s.score,
			Content: s.memo.Content,
			Memo:    s.memo,
		})
	}

	// 按分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// QualityLevel 结果质量等级.
type QualityLevel int

const (
	LowQuality QualityLevel = iota
	MediumQuality
	HighQuality
)

// String 返回质量等级的字符串表示.
func (q QualityLevel) String() string {
	switch q {
	case LowQuality:
		return "low"
	case MediumQuality:
		return "medium"
	case HighQuality:
		return "high"
	default:
		return "unknown"
	}
}

// evaluateQuality 评估结果质量.
func (r *AdaptiveRetriever) evaluateQuality(results []*SearchResult) QualityLevel {
	if len(results) == 0 {
		return LowQuality
	}

	topScore := results[0].Score

	// 判断 1：前2名分数差距大 → 高质量
	if len(results) >= 2 {
		scoreGap := topScore - results[1].Score
		if scoreGap > 0.20 {
			return HighQuality
		}
	}

	// 判断 2：第1名分数很高 → 高质量
	if topScore > 0.90 {
		return HighQuality
	}

	// 判断 3：第1名分数中等 → 中等质量
	if topScore > 0.70 {
		return MediumQuality
	}

	// 否则：低质量
	return LowQuality
}

// mergeResults 合并结果（去重，按分数排序）.
func (r *AdaptiveRetriever) mergeResults(results1, results2 []*SearchResult, topK int) []*SearchResult {
	// 去重（基于 ID）
	seen := make(map[int64]bool)
	merged := make([]*SearchResult, 0)

	for _, result := range results1 {
		if !seen[result.ID] {
			seen[result.ID] = true
			merged = append(merged, result)
		}
	}

	for _, result := range results2 {
		if !seen[result.ID] {
			seen[result.ID] = true
			merged = append(merged, result)
		}
	}

	// 按分数排序
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Score > merged[j].Score
	})

	// 返回 Top K
	return r.truncateResults(merged, topK)
}

// shouldRerank 判断是否需要重排.
func (r *AdaptiveRetriever) shouldRerank(query string, results []*SearchResult) bool {
	// 检查 Reranker 是否启用
	if r.rerankerService == nil || !r.rerankerService.IsEnabled() {
		return false
	}

	// 规则 1：结果少（<5），不需要重排
	if len(results) < 5 {
		return false
	}

	// 规则 2：简单查询，不需要重排
	if r.isSimpleKeywordQuery(query) {
		return false
	}

	// 规则 3：前2名分数差距大（>0.15），不需要重排
	if len(results) >= 2 {
		if results[0].Score-results[1].Score > 0.15 {
			return false
		}
	}

	// 其他情况：需要重排
	return true
}

// isSimpleKeywordQuery 判断是否为简单关键词查询.
func (r *AdaptiveRetriever) isSimpleKeywordQuery(query string) bool {
	// 简单查询特征：
	// 1. 查询短（<10个字符）
	if len(query) < 10 {
		return true
	}

	// 2. 检测是否有疑问词、连词等复杂语法
	complexWords := []string{"如何", "怎么", "为什么", "和", "或者", "但是", "how", "why"}
	for _, word := range complexWords {
		if strings.Contains(query, word) {
			return false
		}
	}

	return true
}

// filterByScore 过滤低分结果.
func (r *AdaptiveRetriever) filterByScore(results []*SearchResult, minScore float32) []*SearchResult {
	if minScore <= 0 {
		return results
	}

	filtered := make([]*SearchResult, 0)
	for _, result := range results {
		if result.Score >= minScore {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// truncateResults 截断结果到指定数量.
func (r *AdaptiveRetriever) truncateResults(results []*SearchResult, limit int) []*SearchResult {
	if limit <= 0 || len(results) <= limit {
		return results
	}
	return results[:limit]
}

// generateRequestID 生成唯一的请求 ID。
func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-only if crypto rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), b)
}
