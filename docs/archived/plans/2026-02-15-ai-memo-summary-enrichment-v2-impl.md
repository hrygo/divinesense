# AI 摘要生成与 Memo 内容增强实施计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 AI 摘要生成、内容格式化功能，建立统一的内容增强 Pipeline，支持双轨制标签增强

**Architecture:** 创建 `ai/enrichment/` Pipeline 框架，`ai/summary/` 摘要服务，`ai/format/` 格式化服务，扩展 store 层支持 memo_summary 和 memo_tags 表

**Tech Stack:** Go, Connect RPC, PostgreSQL, React

---

## 阶段 1: 基础设施（配置加载器 + Pipeline 接口）

### Task S1.1: 创建统一配置加载器

**Files:**
- Create: `ai/configloader/loader.go`

**Step 1: 创建目录和基础结构**

```bash
mkdir -p ai/configloader
```

**Step 2: 编写配置加载器代码**

```go
package configloader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Loader 是通用的 YAML 配置加载器
type Loader struct {
	baseDir string
	cache   sync.Map
}

// NewLoader 创建加载器，baseDir 为项目根目录
func NewLoader(baseDir string) *Loader {
	return &Loader{baseDir: baseDir}
}

// Load 加载单个 YAML 文件到目标结构体
func (l *Loader) Load(subPath string, target any) error {
	absPath := filepath.Join(l.baseDir, subPath)
	data, err := ReadFileWithFallback(absPath)
	if err != nil {
		return fmt.Errorf("load %s: %w", subPath, err)
	}
	return yaml.Unmarshal(data, target)
}

// LoadCached 带缓存的加载（适合不变的 Prompt 配置）
func (l *Loader) LoadCached(subPath string, factory func() any) (any, error) {
	if cached, ok := l.cache.Load(subPath); ok {
		return cached, nil
	}
	target := factory()
	if err := l.Load(subPath, target); err != nil {
		return nil, err
	}
	l.cache.Store(subPath, target)
	return target, nil
}

// LoadDir 批量加载目录下所有 YAML
func (l *Loader) LoadDir(subDir string, factory func(path string) (any, error)) (map[string]any, error) {
	dir := filepath.Join(l.baseDir, subDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", subDir, err)
	}
	results := make(map[string]any)
	for _, entry := range entries {
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".yaml" && filepath.Ext(entry.Name()) != ".yml") {
			continue
		}
		item, err := factory(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", entry.Name(), err)
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		results[name] = item
	}
	return results, nil
}

// ReadFileWithFallback 读取文件，支持可执行文件目录 fallback
func ReadFileWithFallback(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}
	execPath, execErr := os.Executable()
	if execErr != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(filepath.Dir(execPath), path))
}
```

**Step 3: 验证编译**

```bash
go build ./ai/configloader/...
```

**Step 4: Commit**

```bash
git add ai/configloader/
git commit -m "feat(ai): add unified configloader package"
```

---

### Task S1.2: 迁移 orchestrator/prompts.go 使用 configloader

**Files:**
- Modify: `ai/agents/orchestrator/prompts.go:1-50`

**Step 1: 添加 import 和替换调用**

在文件头部添加:
```go
import (
    // ... existing imports
    "github.com/hrygo/divinesense/ai/configloader"
)
```

**Step 2: 修改 LoadPromptConfig 函数**

找到现有的 LoadPromptConfig 函数（约第 50 行），替换为:
```go
func LoadPromptConfig() (*PromptConfig, error) {
	loader := configloader.NewLoader(getBaseDir())
	var cfg PromptConfig
	err := loader.Load("config/orchestrator/prompts.yaml", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func getBaseDir() string {
	// 获取项目根目录
	execPath, _ := os.Executable()
	return filepath.Dir(execPath)
}
```

**Step 3: 验证编译**

```bash
go build ./ai/agents/orchestrator/...
```

**Step 4: Commit**

```bash
git add ai/agents/orchestrator/prompts.go
git commit -m "refactor(orchestrator): migrate to unified configloader"
```

---

### Task S1.3: 创建 Enricher 接口

**Files:**
- Create: `ai/enrichment/enricher.go`

**Step 1: 创建目录和接口定义**

```bash
mkdir -p ai/enrichment
```

**Step 2: 编写 Enricher 接口代码**

```go
package enrichment

import (
	"context"
	"time"
)

// EnrichmentType 标识增强类型
type EnrichmentType string

// Phase 标识执行阶段
type Phase string

const (
	// Pre-save（同步，用户触发）
	EnrichmentFormat EnrichmentType = "format"

	// Post-save（异步，自动触发）
	EnrichmentSummary EnrichmentType = "summary"
	EnrichmentTags   EnrichmentType = "tags"
	EnrichmentTitle  EnrichmentType = "title"
)

const (
	PhasePre  Phase = "pre_save"  // 同步，保存前
	PhasePost Phase = "post_save" // 异步，保存后
)

// MemoContent 是增强器的统一输入
type MemoContent struct {
	MemoID  string
	Content string
	Title   string
	UserID  int32
}

// EnrichmentResult 是单个增强器的输出
type EnrichmentResult struct {
	Type    EnrichmentType
	Success bool
	Data    any
	Error   error
	Latency time.Duration
}

// Enricher 是内容增强器的统一接口
type Enricher interface {
	// Type 返回增强器类型
	Type() EnrichmentType
	// Phase 返回该 Enricher 所属阶段
	Phase() Phase
	// Enrich 执行增强，返回结果
	Enrich(ctx context.Context, content *MemoContent) *EnrichmentResult
}
```

**Step 3: 验证编译**

```bash
go build ./ai/enrichment/...
```

**Step 4: Commit**

```bash
git add ai/enrichment/enricher.go
git commit -m "feat(ai): add Enricher interface for content enrichment"
```

---

### Task S1.4: 创建 Pipeline 编排器

**Files:**
- Create: `ai/enrichment/pipeline.go`
- Create: `ai/enrichment/pipeline_test.go`

**Step 1: 编写 Pipeline 编排器代码**

```go
package enrichment

import (
	"context"
	"sync"
	"time"
)

// Pipeline 编排多个 Enricher
type Pipeline struct {
	enrichers []Enricher
	timeout   time.Duration
}

// NewPipeline 创建增强管线
func NewPipeline(enrichers ...Enricher) *Pipeline {
	return &Pipeline{
		enrichers: enrichers,
		timeout:   30 * time.Second,
	}
}

// EnrichAll 并行执行所有增强器，返回结果集合
func (p *Pipeline) EnrichAll(ctx context.Context, content *MemoContent) map[EnrichmentType]*EnrichmentResult {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	results := make(map[EnrichmentType]*EnrichmentResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, e := range p.enrichers {
		wg.Add(1)
		go func(enricher Enricher) {
			defer wg.Done()
			result := enricher.Enrich(ctx, content)
			mu.Lock()
			results[enricher.Type()] = result
			mu.Unlock()
		}(e)
	}

	wg.Wait()
	return results
}

// EnrichPostSave 执行 Post-save 阶段的增强（异步并行）
func (p *Pipeline) EnrichPostSave(ctx context.Context, content *MemoContent) map[EnrichmentType]*EnrichmentResult {
	var postEnrichers []Enricher
	for _, e := range p.enrichers {
		if e.Phase() == PhasePost {
			postEnrichers = append(postEnrichers, e)
		}
	}
	if len(postEnrichers) == 0 {
		return nil
	}
	// 临时创建只包含 post 阶段 enrichers 的 pipeline
	tmpPipeline := NewPipeline(postEnrichers...)
	return tmpPipeline.EnrichAll(ctx, content)
}

// EnrichOne 执行单个类型的增强
func (p *Pipeline) EnrichOne(ctx context.Context, t EnrichmentType, content *MemoContent) *EnrichmentResult {
	for _, e := range p.enrichers {
		if e.Type() == t {
			return e.Enrich(ctx, content)
		}
	}
	return &EnrichmentResult{Type: t, Success: false, Error: ErrEnricherNotFound}
}

// Errors
var ErrEnricherNotFound = &EnricherNotFoundError{}

type EnricherNotFoundError struct{}

func (e *EnricherNotFoundError) Error() string {
	return "enricher not found"
}
```

**Step 2: 编写测试代码**

```go
package enrichment

import (
	"context"
	"testing"
	"time"
)

// mockEnricher 用于测试
type mockEnricher struct {
	enrichmentType EnrichmentType
	phase          Phase
	latency        time.Duration
}

func (m *mockEnricher) Type() EnrichmentType        { return m.enrichmentType }
func (m *mockEnricher) Phase() Phase                 { return m.phase }
func (m *mockEnricher) Enrich(ctx context.Context, content *MemoContent) *EnrichmentResult {
	time.Sleep(m.latency)
	return &EnrichmentResult{
		Type:    m.enrichmentType,
		Success: true,
		Data:    "mock result",
	}
}

func TestPipeline_EnrichAll(t *testing.T) {
	pipeline := NewPipeline(
		&mockEnricher{enrichmentType: EnrichmentSummary, phase: PhasePost, latency: 10 * time.Millisecond},
		&mockEnricher{enrichmentType: EnrichmentTags, phase: PhasePost, latency: 20 * time.Millisecond},
	)

	content := &MemoContent{
		MemoID:  "test-123",
		Content: "test content",
		Title:   "test title",
		UserID:  1,
	}

	results := pipeline.EnrichAll(context.Background(), content)

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if !results[EnrichmentSummary].Success {
		t.Error("summary enricher should succeed")
	}
	if !results[EnrichmentTags].Success {
		t.Error("tags enricher should succeed")
	}
}

func TestPipeline_EnrichPostSave(t *testing.T) {
	pipeline := NewPipeline(
		&mockEnricher{enrichmentType: EnrichmentFormat, phase: PhasePre, latency: 10 * time.Millisecond},
		&mockEnricher{enrichmentType: EnrichmentSummary, phase: PhasePost, latency: 10 * time.Millisecond},
	)

	content := &MemoContent{MemoID: "test-123", Content: "test", UserID: 1}
	results := pipeline.EnrichPostSave(context.Background(), content)

	// 应该只返回 post-save 阶段的 enricher 结果
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
	if _, ok := results[EnrichmentFormat]; ok {
		t.Error("should not include pre-save enricher in post-save results")
	}
}
```

**Step 3: 运行测试**

```bash
go test ./ai/enrichment/... -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add ai/enrichment/pipeline.go ai/enrichment/pipeline_test.go
git commit -m "feat(ai): add Pipeline orchestrator for content enrichment"
```

---

## 阶段 2: 摘要功能

### Task S2.1: 创建 Summary 接口

**Files:**
- Create: `ai/summary/summarizer.go`

**Step 1: 创建目录和接口**

```bash
mkdir -p ai/summary
```

**Step 2: 编写接口代码**

```go
package summary

import (
	"context"
	"time"
)

// Summarizer 提供笔记摘要能力
type Summarizer interface {
	// Summarize 生成笔记摘要
	Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error)
}

// SummarizeRequest 摘要请求
type SummarizeRequest struct {
	MemoID  string
	Content string
	Title   string
	MaxLen  int // 摘要最大长度（rune），默认 200
}

// SummarizeResponse 摘要响应
type SummarizeResponse struct {
	Summary string
	Source  string        // "llm" | "fallback_first_para" | "fallback_first_sentence" | "fallback_truncate"
	Latency time.Duration
}
```

**Step 3: 验证编译**

```bash
go build ./ai/summary/...
```

**Step 4: Commit**

```bash
git add ai/summary/summarizer.go
git commit -m "feat(ai): add Summarizer interface"
```

---

### Task S2.2: 创建 Fallback 三级降级

**Files:**
- Create: `ai/summary/fallback.go`

**Step 1: 编写 Fallback 代码**

```go
package summary

import (
	"strings"
	"unicode/utf8"
)

// FallbackSummarize 提供三级降级摘要
func FallbackSummarize(req *SummarizeRequest) (*SummarizeResponse, error) {
	maxLen := req.MaxLen
	if maxLen <= 0 {
		maxLen = 200
	}

	// Level 1: 首段提取（最优降级）
	if firstPara := extractFirstParagraph(req.Content); firstPara != "" {
		if utf8.RuneCountInString(firstPara) <= maxLen {
			return &SummarizeResponse{
				Summary: firstPara,
				Source:  "fallback_first_para",
			}, nil
		}
		// 首段超长，截断
		return &SummarizeResponse{
			Summary: truncateRunes(firstPara, maxLen),
			Source:  "fallback_first_para",
		}, nil
	}

	// Level 2: 首句提取
	if firstSentence := extractFirstSentence(req.Content); firstSentence != "" {
		if utf8.RuneCountInString(firstSentence) <= maxLen {
			return &SummarizeResponse{
				Summary: firstSentence,
				Source:  "fallback_first_sentence",
			}, nil
		}
		// 首句超长，截断
		return &SummarizeResponse{
			Summary: truncateRunes(firstSentence, maxLen),
			Source:  "fallback_first_sentence",
		}, nil
	}

	// Level 3: Rune 安全截断（保底）
	return &SummarizeResponse{
		Summary: truncateRunes(req.Content, maxLen),
		Source:  "fallback_truncate",
	}, nil
}

// extractFirstParagraph 提取第一段
func extractFirstParagraph(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// extractFirstSentence 提取第一句
func extractFirstSentence(content string) string {
	// 简单实现：按句号、问号、感叹号分割
	firstLine := extractFirstParagraph(content)
	if firstLine == "" {
		return ""
	}

	endMarkers := []string{".", "。", "?", "？", "!", "！"}
	for _, marker := range endMarkers {
		if idx := strings.Index(firstLine, marker); idx > 0 {
			return firstLine[:idx+len(marker)]
		}
	}
	return firstLine
}

// truncateRunes 安全截断字符串（按 rune 而非 byte）
func truncateRunes(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// runeLen 获取字符串的 rune 长度
func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}
```

**Step 2: 编写测试**

```go
package summary

import (
	"testing"
)

func TestFallbackSummarize_FirstParagraph(t *testing.T) {
	req := &SummarizeRequest{
		Content: "这是第一段内容\n\n这是第二段内容\n\n这是第三段内容",
		MaxLen:  50,
	}

	resp, err := FallbackSummarize(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Source != "fallback_first_para" {
		t.Errorf("expected source 'fallback_first_para', got '%s'", resp.Source)
	}

	if resp.Summary != "这是第一段内容" {
		t.Errorf("expected '这是第一段内容', got '%s'", resp.Summary)
	}
}

func TestFallbackSummarize_Truncate(t *testing.T) {
	req := &SummarizeRequest{
		Content: "这是一段非常非常长的内容，超过了最大长度限制，需要被截断",
		MaxLen:  10,
	}

	resp, err := FallbackSummarize(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Source != "fallback_first_para" {
		t.Errorf("expected source 'fallback_first_para', got '%s'", resp.Source)
	}

	if len([]rune(resp.Summary)) > 10 {
		t.Errorf("summary should be truncated to 10 runes, got %d", len([]rune(resp.Summary)))
	}
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"normal", "hello world", 5, "hello"},
		{"chinese", "你好世界", 2, "你好"},
		{"empty", "", 10, ""},
		{"short", "hi", 10, "hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateRunes(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
```

**Step 3: 运行测试**

```bash
go test ./ai/summary/... -v
```

Expected: PASS

**Step 4: Commit**

```bash
git add ai/summary/fallback.go ai/summary/fallback_test.go
git commit -m "feat(ai): add FallbackSummarize with 3-level degradation"
```

---

### Task S2.3: 创建 Summary LLM 实现

**Files:**
- Create: `ai/summary/summarizer_impl.go`

**Step 1: 编写 LLM 实现**

```go
package summary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
)

// llmSummarizer 使用 LLM 生成摘要
type llmSummarizer struct {
	llm     llm.Service
	timeout time.Duration
}

// NewSummarizer 创建摘要生成器
func NewSummarizer(llmSvc llm.Service) Summarizer {
	return &llmSummarizer{
		llm:     llmSvc,
		timeout: 15 * time.Second,
	}
}

func (s *llmSummarizer) Summarize(ctx context.Context, req *SummarizeRequest) (*SummarizeResponse, error) {
	maxLen := req.MaxLen
	if maxLen <= 0 {
		maxLen = 200
	}

	// 1. 短文本无需摘要
	if runeLen(req.Content) <= maxLen {
		return &SummarizeResponse{
			Summary: req.Content,
			Source:  "original",
		}, nil
	}

	// 2. LLM 不可用时走 Fallback
	if s.llm == nil {
		return FallbackSummarize(req)
	}

	// 3. LLM 生成摘要
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// 构建 prompt
	userPrompt := fmt.Sprintf(`请为以下笔记生成不超过 %d 字的摘要：

%s

请直接返回JSON格式：{"summary": "生成的摘要"}`, maxLen, req.Content)

	messages := []llm.Message{
		llm.SystemPrompt(summarySystemPrompt),
		llm.UserMessage(userPrompt),
	}

	content, stats, err := s.llm.Chat(ctx, messages)
	if err != nil {
		// LLM 失败，降级到 Fallback
		return FallbackSummarize(req)
	}

	// 4. 解析并截断
	summary := parseSummary(content)
	summary = truncateRunes(summary, maxLen)

	return &SummarizeResponse{
		Summary: summary,
		Source:  "llm",
		Latency: stats.TotalDurationMs,
	}, nil
}

// parseSummary 从 LLM 响应中解析摘要
func parseSummary(content string) string {
	// 尝试解析 JSON
	var result struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(content), &result); err == nil && result.Summary != "" {
		return strings.TrimSpace(result.Summary)
	}

	// 如果不是 JSON，尝试提取 "summary": 后的内容
	if idx := strings.Index(content, `"summary"`); idx >= 0 {
		start := strings.Index(content[idx:], ":") + idx + 1
		end := strings.Index(content[start:], "}")
		if end > 0 {
			return strings.Trim(content[start:start+end], `" `)
		}
	}

	// 直接返回清理后的内容
	return strings.TrimSpace(content)
}

const summarySystemPrompt = `你是一个专业的笔记摘要助手。你的任务是根据笔记原文，生成一段精炼的摘要。

要求：
1. 摘要长度不超过指定字数
2. 保留笔记的核心观点和关键信息
3. 使用与原文一致的语言
4. 不要添加原文没有的观点
5. 直接输出摘要文本，不要添加"摘要："等前缀
6. 返回JSON格式：{"summary": "生成的摘要"}`
```

**Step 2: 验证编译**

```bash
go build ./ai/summary/...
```

**Step 3: Commit**

```bash
git add ai/summary/summarizer_impl.go
git commit -m "feat(ai): add LLM implementation for Summarizer"
```

---

### Task S2.4: 创建 Summary Prompt 配置

**Files:**
- Create: `config/prompts/summary.yaml`

**Step 1: 创建配置文件**

```yaml
# config/prompts/summary.yaml
name: summary
version: "1.0"

system_prompt: |
  你是一个专业的笔记摘要助手。你的任务是根据笔记原文，生成一段精炼的摘要。

  要求：
  1. 摘要长度不超过指定字数
  2. 保留笔记的核心观点和关键信息
  3. 使用与原文一致的语言（中文笔记用中文摘要，英文笔记用英文摘要）
  4. 不要添加原文没有的观点
  5. 如果笔记是列表/清单格式，摘要应概述主题和要点数量
  6. 直接输出摘要文本，不要添加"摘要："等前缀

  请直接返回JSON格式：{"summary": "生成的摘要"}

user_prompt_template: |
  请为以下笔记生成不超过 {{.MaxLen}} 字的摘要：

  {{.Content}}

params:
  max_tokens: 300
  temperature: 0.3
  timeout_seconds: 15
  input_truncate_chars: 3000
```

**Step 2: Commit**

```bash
git add config/prompts/summary.yaml
git commit -m "feat(config): add summary prompt configuration"
```

---

## 阶段 2b: 格式化功能

### Task S2b.1: 创建 Format 接口

**Files:**
- Create: `ai/format/formatter.go`

**Step 1: 创建目录和接口**

```bash
mkdir -p ai/format
```

**Step 2: 编写接口代码**

```go
package format

import (
	"context"
	"time"
)

// Formatter 将随意输入的文本格式化为标准 Markdown
type Formatter interface {
	Format(ctx context.Context, req *FormatRequest) (*FormatResponse, error)
}

type FormatRequest struct {
	Content string // 用户原始输入
	UserID  int32
}

type FormatResponse struct {
	Formatted string        // 格式化后的 Markdown 内容
	Changed   bool          // 内容是否有变化
	Source    string        // "llm" | "passthrough"
	Latency  time.Duration
}
```

**Step 3: 验证编译**

```bash
go build ./ai/format/...
```

**Step 4: Commit**

```bash
git add ai/format/formatter.go
git commit -m "feat(ai): add Formatter interface"
```

---

### Task S2b.2: 创建 Format LLM 实现

**Files:**
- Create: `ai/format/formatter_impl.go`

**Step 1: 编写 LLM 实现**

```go
package format

import (
	"context"
	"strings"
	"time"

	"github.com/hrygo/divinesense/ai/core/llm"
)

type llmFormatter struct {
	llm     llm.Service
	timeout time.Duration
}

func NewFormatter(llmSvc llm.Service) Formatter {
	return &llmFormatter{
		llm:     llmSvc,
		timeout: 10 * time.Second,
	}
}

func (f *llmFormatter) Format(ctx context.Context, req *FormatRequest) (*FormatResponse, error) {
	start := time.Now()

	// 1. 已经是合格 Markdown 的短文本，直接跳过
	if isWellFormatted(req.Content) {
		return &FormatResponse{
			Formatted: req.Content,
			Changed:   false,
			Source:    "passthrough",
			Latency:  time.Since(start),
		}, nil
	}

	// 2. LLM 不可用时直接放行
	if f.llm == nil {
		return &FormatResponse{
			Formatted: req.Content,
			Changed:   false,
			Source:    "passthrough",
			Latency:  time.Since(start),
		}, nil
	}

	// 3. 调用 LLM 格式化
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	userPrompt := "请将以下内容整理为标准 Markdown 格式：\n\n" + req.Content

	messages := []llm.Message{
		llm.SystemPrompt(formatSystemPrompt),
		llm.UserMessage(userPrompt),
	}

	content, _, err := f.llm.Chat(ctx, messages)
	if err != nil {
		// LLM 失败不阻塞，原样放行
		return &FormatResponse{
			Formatted: req.Content,
			Changed:   false,
			Source:    "passthrough",
			Latency:  time.Since(start),
		}, nil
	}

	formatted := parseFormattedContent(content)
	return &FormatResponse{
		Formatted: formatted,
		Changed:   formatted != req.Content,
		Source:    "llm",
		Latency:  time.Since(start),
	}, nil
}

// isWellFormatted 判断是否已经是合格 Markdown
func isWellFormatted(content string) bool {
	if len(content) < 50 {
		return false
	}
	lines := strings.Split(content, "\n")
	mdMarkers := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "- ") ||
			strings.HasPrefix(trimmed, "* ") ||
			strings.HasPrefix(trimmed, "```") ||
			strings.HasPrefix(trimmed, "1. ") {
			mdMarkers++
		}
	}
	return mdMarkers >= 2
}

// parseFormattedContent 解析 LLM 返回的格式化内容
func parseFormattedContent(content string) string {
	// 清理可能的代码块标记
	content = strings.TrimSpace(content)
	content = strings.Trim(content, "```markdown")
	content = strings.Trim(content, "```")
	content = strings.TrimSpace(content)
	return content
}

const formatSystemPrompt = `你是一个笔记格式化助手。将用户随意输入的内容整理为结构清晰的 Markdown 格式。

规则：
1. 保持原文含义完全不变，不添加、不删除任何信息
2. 合理使用 Markdown 标记：标题(#)、列表(-)、加粗(**)、代码块(```)
3. 如果内容包含多个主题，使用标题分隔
4. 如果内容是清单/列表形式，转为 Markdown 列表
5. 如果内容已经格式良好，原样返回
6. 不要添加额外的标题或总结
7. 直接返回格式化后的 Markdown，不要包裹在 JSON 或代码块中
8. 如果原文是英文，使用英文标点；如果是中文，使用中文标点`
```

**Step 2: 验证编译**

```bash
go build ./ai/format/...
```

**Step 3: Commit**

```bash
git add ai/format/formatter_impl.go
git commit -m "feat(ai): add LLM implementation for Formatter"
```

---

### Task S2b.3: 创建 Format Prompt 配置

**Files:**
- Create: `config/prompts/format.yaml`

**Step 1: 创建配置文件**

```yaml
# config/prompts/format.yaml
name: format
version: "1.0"

system_prompt: |
  你是一个笔记格式化助手。将用户随意输入的内容整理为结构清晰的 Markdown 格式。

  规则：
  1. 保持原文含义完全不变，不添加、不删除任何信息
  2. 合理使用 Markdown 标记：标题(#)、列表(-)、加粗(**)、代码块(```)
  3. 如果内容包含多个主题，使用标题分隔
  4. 如果内容是清单/清单形式，转为 Markdown 列表
  5. 如果内容已经格式良好，原样返回
  6. 不要添加额外的标题或总结

  直接返回格式化后的 Markdown，不要包裹在 JSON 或代码块中。

user_prompt_template: |
  请将以下内容整理为标准 Markdown 格式：

  {{.Content}}

params:
  max_tokens: 2000
  temperature: 0.1
  timeout_seconds: 10
  input_truncate_chars: 5000
```

**Step 2: Commit**

```bash
git add config/prompts/format.yaml
git commit -m "feat(config): add format prompt configuration"
```

---

### Task S2b.4: 前端 Format 按钮集成

**Files:**
- Modify: `web/src/components/MemoEditor/components/AIFormatButton.tsx`
- Modify: `web/src/hooks/useAIQueries.ts`

**Step 1: 检查现有 AIFormatButton 实现**

```bash
cat web/src/components/MemoEditor/components/AIFormatButton.tsx
```

**Step 2: 添加 format API 调用**

在 `useAIQueries.ts` 中添加:

```typescript
export function useFormatContent() {
  return useMutation({
    mutationFn: async (content: string) => {
      const request = create(FormatRequestSchema, { content });
      const response = await aiServiceClient.format(request);
      return response;
    },
  });
}
```

**Step 3: 修改 AIFormatButton 使用 API**

在按钮点击时调用 `useFormatContent` hook

**Step 4: 验证编译**

```bash
cd web && pnpm build
```

**Step 5: Commit**

```bash
git add web/src/components/MemoEditor/components/AIFormatButton.tsx web/src/hooks/useAIQueries.ts
git commit -m "feat(web): integrate Format button with API"
```

---

### Task S2b.5: Format API 端点

**Files:**
- Modify: `proto/api/v1/ai_service.proto`
- Modify: `server/router/api/v1/ai_service_semantic.go`

**Step 1: 添加 proto 定义**

```protobuf
// proto/api/v1/ai_service.proto 在 SuggestTags rpc 之后添加

rpc Format(FormatRequest) returns (FormatResponse) {
  option (google.api.http) = {
    post: "/api/v1/ai/format"
    body: "*"
  };
}

message FormatRequest {
  string content = 1;
}

message FormatResponse {
  string formatted = 1;
  bool changed = 2;
  string source = 3;
}
```

**Step 2: 重新生成 proto 代码**

```bash
make proto
```

**Step 3: 实现 Handler**

在 `server/router/api/v1/ai_service_semantic.go` 中添加:

```go
func (s *AIService) Format(ctx context.Context, req *v1pb.FormatRequest) (*v1pb.FormatResponse, error) {
    formatter := format.NewFormatter(s.LLM)
    resp, err := formatter.Format(ctx, &format.FormatRequest{
        Content: req.GetContent(),
        UserID:  s.CurrentUserID,
    })
    if err != nil {
        return nil, err
    }
    return &v1pb.FormatResponse{
        Formatted: resp.Formatted,
        Changed:   resp.Changed,
        Source:    resp.Source,
    }, nil
}
```

**Step 4: 验证编译**

```bash
go build ./server/...
```

**Step 5: Commit**

```bash
git add proto/api/v1/ai_service.proto server/router/api/v1/ai_service_semantic.go
git commit -m "feat(api): add Format endpoint"
```

---

## 阶段 3: 存储层

### Task S3.1: 创建 DB 迁移 (memo_summary)

**Files:**
- Create: `store/migration/postgres/XXXXXX_memo_summary.sql`

**Step 1: 创建迁移文件**

```sql
-- store/migration/postgres/XXXXXX_memo_summary.sql
CREATE TABLE IF NOT EXISTS memo_summary (
    memo_id    INTEGER PRIMARY KEY REFERENCES memo(id) ON DELETE CASCADE,
    summary    TEXT NOT NULL,
    source     VARCHAR(32) NOT NULL DEFAULT 'fallback_truncate',
    version    INTEGER NOT NULL DEFAULT 1,
    created_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_memo_summary_source ON memo_summary(source);
```

**Step 2: 运行迁移**

```bash
make db-migrate
```

**Step 3: Commit**

```bash
git add store/migration/postgres/
git commit -m "feat(db): add memo_summary table"
```

---

### Task S3.2: 创建 Store 层 CRUD

**Files:**
- Modify: `store/driver.go`
- Create: `store/memo_summary.go`

**Step 1: 扩展 Driver 接口**

在 `store/driver.go` 的 Driver 接口中添加:

```go
// Summary related
UpsertMemoSummary(ctx context.Context, upsert *UpsertMemoSummary) error
GetMemoSummary(ctx context.Context, memoID int32) (*MemoSummary, error)
BatchGetMemoSummaries(ctx context.Context, memoIDs []int32) (map[int32]*MemoSummary, error)
```

**Step 2: 创建 store/memo_summary.go**

```go
package store

import (
	"context"
	"database/sql"
)

// MemoSummary 表示笔记摘要
type MemoSummary struct {
	MemoID    int32
	Summary   string
	Source    string
	Version   int
	CreatedTs interface{}
	UpdatedTs interface{}
}

// UpsertMemoSummary 更新或插入摘要
func (s *Store) UpsertMemoSummary(ctx context.Context, upsert *UpsertMemoSummary) error {
	query := `
		INSERT INTO memo_summary (memo_id, summary, source, version, updated_ts)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (memo_id) DO UPDATE SET
			summary = EXCLUDED.summary,
			source = EXCLUDED.source,
			version = EXCLUDED.version + 1,
			updated_ts = NOW()
	`
	_, err := s.db.ExecContext(ctx, query, upsert.MemoID, upsert.Summary, upsert.Source, 1)
	return err
}

// GetMemoSummary 获取摘要
func (s *Store) GetMemoSummary(ctx context.Context, memoID int32) (*MemoSummary, error) {
	query := `
		SELECT memo_id, summary, source, version, created_ts, updated_ts
		FROM memo_summary WHERE memo_id = $1
	`
	var summary MemoSummary
	err := s.db.QueryRowContext(ctx, query, memoID).Scan(
		&summary.MemoID, &summary.Summary, &summary.Source,
		&summary.Version, &summary.CreatedTs, &summary.UpdatedTs,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &summary, err
}

// BatchGetMemoSummaries 批量获取摘要
func (s *Store) BatchGetMemoSummaries(ctx context.Context, memoIDs []int32) (map[int32]*MemoSummary, error) {
	if len(memoIDs) == 0 {
		return make(map[int32]*MemoSummary), nil
	}
	query := `
		SELECT memo_id, summary, source, version, created_ts, updated_ts
		FROM memo_summary WHERE memo_id = ANY($1)
	`
	rows, err := s.db.QueryContext(ctx, query, memoIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[int32]*MemoSummary)
	for rows.Next() {
		var summary MemoSummary
		if err := rows.Scan(&summary.MemoID, &summary.Summary, &summary.Source,
			&summary.Version, &summary.CreatedTs, &summary.UpdatedTs); err != nil {
			return nil, err
		}
		results[summary.MemoID] = &summary
	}
	return results, rows.Err()
}
```

**Step 3: 添加 UpsertMemoSummary 结构体**

```go
// 在 store/memo_summary.go 添加
type UpsertMemoSummary struct {
	MemoID  int32
	Summary string
	Source  string
}
```

**Step 4: 验证编译**

```bash
go build ./store/...
```

**Step 5: Commit**

```bash
git add store/driver.go store/memo_summary.go
git commit -m "feat(store): add memo_summary CRUD operations"
```

---

## 阶段 4: Pipeline 集成

### Task S4.1: 创建 Tags Enricher 适配器

**Files:**
- Create: `ai/tags/enricher_adapter.go`

**Step 1: 编写适配器**

```go
package tags

import (
	"context"
	"time"

	"github.com/hrygo/divinesense/ai/enrichment"
)

// TagsEnricher 将 TagSuggester 适配为 Enricher 接口
type TagsEnricher struct {
	suggester TagSuggester
	maxTags   int
}

func NewEnricher(suggester TagSuggester, maxTags int) enrichment.Enricher {
	return &TagsEnricher{
		suggester: suggester,
		maxTags:   maxTags,
	}
}

func (e *TagsEnricher) Type() enrichment.EnrichmentType {
	return enrichment.EnrichmentTags
}

func (e *TagsEnricher) Phase() enrichment.Phase {
	return enrichment.PhasePost
}

func (e *TagsEnricher) Enrich(ctx context.Context, content *enrichment.MemoContent) *enrichment.EnrichmentResult {
	start := time.Now()

	resp, err := e.suggester.Suggest(ctx, &SuggestRequest{
		Content: content.Content,
		Title:   content.Title,
		MaxTags: e.maxTags,
		UseLLM:  true,
		UserID:  content.UserID,
	})

	if err != nil {
		return &enrichment.EnrichmentResult{
			Type:    enrichment.EnrichmentTags,
			Success: false,
			Error:   err,
			Latency: time.Since(start),
		}
	}

	// 转换为字符串数组
	tags := make([]string, len(resp.Tags))
	for i, tag := range resp.Tags {
		tags[i] = tag.Name
	}

	return &enrichment.EnrichmentResult{
		Type:    enrichment.EnrichmentTags,
		Success: true,
		Data:    tags,
		Latency: time.Since(start),
	}
}
```

**Step 2: 验证编译**

```bash
go build ./ai/tags/...
```

**Step 3: Commit**

```bash
git add ai/tags/enricher_adapter.go
git commit -m "feat(ai): add TagsEnricher adapter for Pipeline"
```

---

### Task S4.2: 重构 Title Generator 添加适配接口

**Files:**
- Modify: `ai/title_generator.go`

**Step 1: 添加适配方法**

在 `ai/title_generator.go` 中添加:

```go
// ToEnricher 将 TitleGenerator 转换为 Enricher 接口
func (g *TitleGenerator) ToEnricher() enrichment.Enricher {
	return &titleEnricher{generator: g}
}

type titleEnricher struct {
	generator *TitleGenerator
}

func (e *titleEnricher) Type() enrichment.EnrichmentType {
	return enrichment.EnrichmentTitle
}

func (e *titleEnricher) Phase() enrichment.Phase {
	return enrichment.PhasePost
}

func (e *titleEnricher) Enrich(ctx context.Context, content *enrichment.MemoContent) *enrichment.EnrichmentResult {
	start := time.Now()

	title, err := e.generator.GenerateTitle(ctx, content.Content)
	if err != nil {
		return &enrichment.EnrichmentResult{
			Type:    enrichment.EnrichmentTitle,
			Success: false,
			Error:   err,
			Latency: time.Since(start),
		}
	}

	return &enrichment.EnrichmentResult{
		Type:    enrichment.EnrichmentTitle,
		Success: true,
		Data:    title,
		Latency: time.Since(start),
	}
}
```

**Step 2: 添加缺失方法**

如果 `GenerateTitle` 方法不存在，需要添加

**Step 3: 验证编译**

```bash
go build ./ai/...
```

**Step 4: Commit**

```bash
git add ai/title_generator.go
git commit -m "refactor(ai): add Enricher adapter for TitleGenerator"
```

---

### Task S4.3: 创建异步触发逻辑

**Files:**
- Modify: `server/router/api/v1/memo_service.go`

**Step 1: 在 Memo 保存后触发 Pipeline**

找到创建/更新 Memo 的 handler，添加:

```go
// 在 CreateMemo 成功后
go func() {
    defer func() { recover() }()

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    pipeline := enrichment.NewPipeline(
        summaryEnricher,
        tagsEnricher,
        titleEnricher,
    )

    results := pipeline.EnrichPostSave(ctx, &enrichment.MemoContent{
        MemoID:  fmt.Sprintf("%d", memo.ID),
        Content: memo.Content,
        Title:   memo.Title,
        UserID:  userID,
    })

    // 存储结果
    if summaryResult, ok := results[enrichment.EnrichmentSummary]; ok && summaryResult.Success {
        _ = s.Store.UpsertMemoSummary(ctx, &store.UpsertMemoSummary{
            MemoID:  memo.ID,
            Summary: summaryResult.Data.(string),
            Source:  "llm",
        })
    }
}()
```

**Step 2: 验证编译**

```bash
go build ./server/...
```

**Step 3: Commit**

```bash
git add server/router/api/v1/memo_service.go
git commit -m "feat(ai): trigger Pipeline after memo save"
```

---

### Task S4.4: Summary API 集成

**Files:**
- Modify: `proto/api/v1/memo_service.proto`
- Modify: `server/router/api/v1/memo_service.go`

**Step 1: 添加 proto 定义**

```protobuf
// 获取 Memo 摘要
rpc GetSummary(GetSummaryRequest) returns (GetSummaryResponse) {
  option (google.api.http) = {
    get: "/api/v1/memos/{uid}/summary"
  };
}

message GetSummaryRequest {
  string uid = 1;
}

message GetSummaryResponse {
  string summary = 1;
  string source = 2;
}
```

**Step 2: 实现 Handler**

```go
func (s *MemoService) GetSummary(ctx context.Context, req *v1pb.GetSummaryRequest) (*v1pb.GetSummaryResponse, error) {
    memoID, err := strconv.ParseInt(req.GetUid(), 10, 32)
    if err != nil {
        return nil, err
    }

    summary, err := s.Store.GetMemoSummary(ctx, int32(memoID))
    if err != nil {
        return nil, err
    }

    if summary == nil {
        return &v1pb.GetSummaryResponse{}, nil
    }

    return &v1pb.GetSummaryResponse{
        Summary: summary.Summary,
        Source:  summary.Source,
    }, nil
}
```

**Step 3: 验证编译**

```bash
go build ./server/...
```

**Step 4: Commit**

```bash
git add proto/api/v1/memo_service.proto server/router/api/v1/memo_service.go
git commit -m "feat(api): add GetSummary endpoint"
```

---

## 阶段 5: 标签双轨制

### Task S5.1: 创建 DB 迁移 (memo_tags)

**Files:**
- Create: `store/migration/postgres/XXXXXX_memo_tags.sql`

**Step 1: 创建迁移文件**

```sql
CREATE TABLE IF NOT EXISTS memo_tags (
    id         SERIAL PRIMARY KEY,
    memo_id    INTEGER NOT NULL REFERENCES memo(id) ON DELETE CASCADE,
    tag        VARCHAR(64) NOT NULL,
    source     VARCHAR(32) NOT NULL DEFAULT 'pipeline',
    confidence FLOAT,
    created_ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(memo_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_memo_tags_memo_id ON memo_tags(memo_id);
```

**Step 2: Commit**

```bash
git add store/migration/postgres/
git commit -m "feat(db): add memo_tags table"
```

---

### Task S5.2: 创建 Store 层 CRUD

**Files:**
- Create: `store/memo_tags.go`

**Step 1: 创建 CRUD 实现**

```go
package store

import (
	"context"
	"database/sql"
)

// MemoTag 表示笔记标签
type MemoTag struct {
	ID         int32
	MemoID     int32
	Tag        string
	Source     string
	Confidence *float64
	CreatedTs  interface{}
}

// UpsertMemoTags 批量更新标签
func (s *Store) UpsertMemoTags(ctx context.Context, upsert *UpsertMemoTags) error {
	if len(upsert.Tags) == 0 {
		return nil
	}

	// 先删除旧的
	_, err := s.db.ExecContext(ctx, "DELETE FROM memo_tags WHERE memo_id = $1 AND source = $2",
		upsert.MemoID, upsert.Source)
	if err != nil {
		return err
	}

	// 批量插入新的
	for _, tag := range upsert.Tags {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO memo_tags (memo_id, tag, source, confidence)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT DO NOTHING
		`, upsert.MemoID, tag.Name, upsert.Source, tag.Confidence)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListMemoTags 获取标签列表
func (s *Store) ListMemoTags(ctx context.Context, memoID int32) ([]*MemoTag, error) {
	query := `
		SELECT id, memo_id, tag, source, confidence, created_ts
		FROM memo_tags WHERE memo_id = $1 ORDER BY confidence DESC
	`
	rows, err := s.db.QueryContext(ctx, query, memoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*MemoTag
	for rows.Next() {
		var tag MemoTag
		if err := rows.Scan(&tag.ID, &tag.MemoID, &tag.Tag, &tag.Source,
			&tag.Confidence, &tag.CreatedTs); err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}
	return tags, rows.Err()
}
```

**Step 2: 添加结构体**

```go
type UpsertMemoTags struct {
	MemoID int32
	Tags   []TagItem
	Source string
}

type TagItem struct {
	Name       string
	Confidence *float64
}
```

**Step 3: 验证编译**

```bash
go build ./store/...
```

**Step 4: Commit**

```bash
git add store/memo_tags.go
git commit -m "feat(store): add memo_tags CRUD operations"
```

---

### Task S5.3: 前端侧边栏展示推荐标签

**Files:**
- Create/Modify: `web/src/components/MemoDetail/` 相关组件

**Step 1: 创建 TagsSidebar 组件**

```tsx
// 组件逻辑
const { data: suggestedTags } = useQuery({
  queryKey: ['memoTags', memoId],
  queryFn: () => fetchMemoTags(memoId),
});

// 渲染标签
{suggestedTags?.map(tag => (
  <TagBadge onClick={() => adoptTag(tag)}>{tag.name}</TagBadge>
))}
```

**Step 2: 集成到 MemoDetail 页面**

**Step 3: 验证编译**

```bash
cd web && pnpm build
```

**Step 4: Commit**

```bash
git add web/src/components/MemoDetail/
git commit -m "feat(web): add suggested tags sidebar"
```

---

## 阶段 6: 测试验收

### Task S6.1: 单元测试

**Step 1: 运行所有单元测试**

```bash
go test ./ai/enrichment/... -v -count=1
go test ./ai/summary/... -v -count=1
go test ./ai/format/... -v -count=1
go test ./store/... -v -count=1
```

**Step 2: 修复失败的测试**

**Step 3: Commit**

```bash
git add .
git commit -m "test: add unit tests for enrichment features"
```

---

### Task S6.2: 集成测试

**Step 1: 运行集成测试**

```bash
go test ./server/... -v -count=1 -tags=integration
```

**Step 2: 修复失败的测试**

**Step 3: Commit**

---

### Task S6.3: E2E 测试

**Step 1: 运行 E2E 测试**

```bash
cd web && pnpm test:e2e
```

**Step 2: 修复失败的测试**

**Step 3: 最终验证**

```bash
make check-all
```

---

## 执行方式选择

**Plan complete and saved to `docs/plans/2026-02-15-ai-memo-summary-enrichment-v2.md`.**

Two execution options:

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
