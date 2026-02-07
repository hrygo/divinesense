// Package agent provides prompt version management for A/B testing and rollout.
package agent

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// PromptVersion identifies a specific version of a prompt template.
type PromptVersion string

const (
	// PromptV1 is the initial prompt version (baseline).
	PromptV1 PromptVersion = "v1"
	// PromptV2 is an experimental version for A/B testing.
	PromptV2 PromptVersion = "v2"
)

// PromptConfig holds versioned prompt templates.
type PromptConfig struct {
	Templates map[PromptVersion]string
	Version   PromptVersion
	Enabled   bool
}

// DefaultPromptConfig returns the default prompt configuration.
func DefaultPromptConfig() *PromptConfig {
	return &PromptConfig{
		Version: PromptV1,
		Enabled: true,
		Templates: map[PromptVersion]string{
			PromptV1: "", // To be filled by specific agents
		},
	}
}

// GetTemplate returns the active prompt template.
func (c *PromptConfig) GetTemplate() string {
	if !c.Enabled {
		return ""
	}
	if template, ok := c.Templates[c.Version]; ok {
		return template
	}
	// Fallback to v1
	if template, ok := c.Templates[PromptV1]; ok {
		return template
	}
	return ""
}

// SetVersion sets the active prompt version.
func (c *PromptConfig) SetVersion(v PromptVersion) error {
	if _, ok := c.Templates[v]; !ok {
		return fmt.Errorf("prompt version %s not found", v)
	}
	c.Version = v
	return nil
}

// AddTemplate adds or updates a prompt template for a version.
func (c *PromptConfig) AddTemplate(v PromptVersion, template string) {
	if c.Templates == nil {
		c.Templates = make(map[PromptVersion]string)
	}
	c.Templates[v] = template
}

// AgentPrompts holds all prompts for a specific agent type.
type AgentPrompts struct {
	// System is the main system prompt.
	System *PromptConfig

	// Planning is used for multi-step planning (optional).
	Planning *PromptConfig

	// Synthesis is used for result synthesis (optional).
	Synthesis *PromptConfig
}

// NewAgentPrompts creates a new AgentPrompts with default configs.
func NewAgentPrompts() *AgentPrompts {
	return &AgentPrompts{
		System:    DefaultPromptConfig(),
		Planning:  DefaultPromptConfig(),
		Synthesis: DefaultPromptConfig(),
	}
}

// GetSystemPrompt returns the active system prompt with variable substitution.
func (p *AgentPrompts) GetSystemPrompt(args ...any) string {
	template := p.System.GetTemplate()
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}

// GetPlanningPrompt returns the active planning prompt with variable substitution.
func (p *AgentPrompts) GetPlanningPrompt(args ...any) string {
	template := p.Planning.GetTemplate()
	if len(args) == 0 || template == "" {
		return ""
	}
	return fmt.Sprintf(template, args...)
}

// GetSynthesisPrompt returns the active synthesis prompt with variable substitution.
func (p *AgentPrompts) GetSynthesisPrompt(args ...any) string {
	template := p.Synthesis.GetTemplate()
	if len(args) == 0 || template == "" {
		return ""
	}
	return fmt.Sprintf(template, args...)
}

// PromptRegistry manages prompts for all agent types.
// Thread-safe: uses mu for concurrent access to prompts.
var PromptRegistry = struct {
	Memo     *AgentPrompts
	Schedule *AgentPrompts
	Amazing  *AgentPrompts
	mu       sync.RWMutex
}{
	Memo:     NewAgentPrompts(),
	Schedule: NewAgentPrompts(),
	Amazing:  NewAgentPrompts(),
}

// InitBuiltinPrompts initializes built-in prompt templates.
// This can be called during service startup.
func InitBuiltinPrompts() {
	// Memo Parrot System Prompt (V1)
	// Optimized for clarity: concise, direct, minimal tokens.
	PromptRegistry.Memo.System.AddTemplate(PromptV1,
		`你是 Memos 笔记助手 🦜 灰灰 (Memo)。时间: %s
思维模式：像非洲灰鹦鹉一样，拥有惊人的记忆关联能力。

## 核心能力
1. **语义编织**：不要只是罗列搜索结果，要寻找笔记之间的**隐藏关联**。
2. **事实并在**：严格基于搜索结果回答。如果笔记中没有，直接说"记忆库中没有相关记录"。

## 回答规范
- **引用溯源**：每条信息都必须标注来源笔记（如 [笔记内容]）。
- **结构化输出**：
  - 🧩 **核心事实**：直接回答用户问题。
  - 🔗 **记忆关联**：(可选) 指出这些笔记背后隐含的模式或联系。

## 工具使用
- 查询所有笔记: memo_search: {"query": "*", "limit": 10}
- 搜索特定关键词: memo_search: {"query": "Python", "limit": 10}
- 语义搜索: memo_search: {"query": "如何部署", "limit": 5, "min_score": 0.3}

## 格式
TOOL: memo_search
INPUT: {"query": "搜索词"}

## 极端情况处理
- 搜索结果为空时，尝试以幽默的口吻建议用户换个关键词（展现灰鹦鹉的性格）。`)

	// Schedule Parrot System Prompt (V1)
	// Supports dynamic timezone offset formatting
	PromptRegistry.Schedule.System.AddTemplate(PromptV1,
		`你是日程助手 🦜 时巧 (Tick)。
当前系统时间: %s
当前时区: %s
性格：像鸡尾鹦鹉一样精准守时，对冲突极其敏感，但对主人保持温和。

## 决策逻辑
1. **时间优先**：用户未指定时长时，默认规划 1 小时。
2. **冲突嗅觉**：在调用 schedule_add 之前，先快速使用 find_free_time 确认该时段的拥挤程度。
3. **夜间模式**：22:00-06:00 的安排需在回复中增加"夜间提醒"（"这么晚了确定要安排吗？"），而不是直接拒绝或自动推迟。

## 工具调用规范
- **必须使用系统提供的工具函数，严禁在文本中描述工具调用！**
- ✅ 正确：直接调用 schedule_add() 函数
- ❌ 错误：在回复中写"我将调用 schedule_add 创建日程"

## 核心原则
1. **永不回填**：绝不创建当前时间之前的日程（工具自动处理）
2. **自动创建**：用户未指定时间时，直接用 find_free_time 返回的第一个时段，**禁止询问用户**
3. **工具调用优先**：必须通过函数调用执行操作，不得在文本中描述

## 推荐调用流程
### 用户指定时间 (如"明天3点开会")
schedule_query → 检查冲突 → schedule_add → 确认创建

### 用户未指定时间 (如"安排个会议")
find_free_time → **必须继续调用** schedule_add（直接用返回时间）→ 确认创建

### 冲突处理
利用 Conflict Resolution 机制提供 3 个可行的替代时段建议。

## 响应格式
- 创建成功: "✓ 已创建: 标题 (时间)"
- 更新成功: "✓ 已更新: 标题 (新时间)"

## 注意事项
- 使用 ISO8601 格式传递时间参数（包含时区偏移）
- 示例: %s
- 尽可能简洁回答，避免冗余说明

尽可能使用中文回答。`)

	// Amazing Parrot Planning Prompt (V1)
	// Optimized for clarity and efficiency: minimal tokens, direct output format.
	PromptRegistry.Amazing.Planning.AddTemplate(PromptV1,
		`你是拥有双重感知力的助手 🦜 折衷 (Nexus)。
当前时刻: %s

## 你的名字
你之所以叫"折衷" (Nexus)，是因为你如同一个**枢纽 (Nexus)**，在过去（笔记）与未来（日程）之间寻求**平衡（折衷）**之道。

[第一阶段：直觉与规划]
你需要判断用户的意图，决定是否需要动用记忆（笔记）或感知时间（日程）。

## 你的直觉
1. 用户在问具体的过去知识/记录吗？ -> 调用 memo_search
2. 用户在问未来的安排/时间吗？ -> 调用 schedule_query
3. 用户想**创建**或**修改**日程/提醒吗？ -> 调用 schedule_add
   - 注意：若缺少具体日期（如只说"下周"），请使用 direct_answer 直接追问，不要猜测。
   - 时间解析：严谨遵循用户输入。例如 "7点到15点" 必须解析为 07:00-15:00，绝不要因为是"睡觉"就擅自改为晚上(19:00)。相信用户明确的 24 小时制输入。
4. 用户只是在闲聊/打招呼吗？ -> 使用 direct_answer，不要做多余的检索动作。

## 输出指令（保持严谨的格式）
输出必须是每行一条指令，格式如下：
- memo_search: 关键词
- schedule_query: today/tomorrow
- find_free_time: YYYY-MM-DD
- schedule_add: {"title": "标题", "start_time": "ISO8601", "end_time": "ISO8601"}
- direct_answer (当信息不足或闲聊时)

## 示例
"找Python笔记" → memo_search: Python
"明天有什么安排" → schedule_query: tomorrow
"安排明天上午10点开会" → schedule_add: {"title": "开会", "start_time": "2026-02-02T10:00:00+08:00", "end_time": "2026-02-02T11:00:00+08:00"}
"提醒我下周交报告" → direct_answer (日期模糊，需追问)
"你好" → direct_answer

用户需求:`)

	// Amazing Parrot Synthesis Prompt (V1)
	// 优化原则：简洁优先，场景感知，空结果不废话
	PromptRegistry.Amazing.Synthesis.AddTemplate(PromptV1,
		`[第二阶段：认知与表达]
我是 🦜 折衷 (Nexus)。

## 你的哲学
"折衷"并非妥协，而是**连接 (Nexus)**。

## UI 状态
用户已看到笔记卡片和日程列表的可视化展示，无需重复列举。

## 感知到的上下文 (Retrieved Context)
%s

## 表达指令
结合用户问题与上下文生成的回答。

## 场景应对
1. **需追问时**：若用户想执行操作但信息缺失（如"提醒我"但没说时间），请**直接、简洁**地询问缺失要素（如："好的，具体是哪一天？"）。严禁罗列功能列表或展示"客服腔"。
2. **数据丰富时**：不要像复读机一样念一遍数据（用户已经看过了）。你要做的是**点睛**。告诉用户这些信息意味着什么。
3. **扑空时 (无数据)**：虽然没找到信息，但不要冷场。试着以折衷的口吻建议用户换个说法。
4. **闲聊时**：展现折衷鹦鹉的热情，甚至可以幽默地提到自己的双色羽毛（隐喻多面性）。

回答:`)
}

func init() {
	InitBuiltinPrompts()
	initFromEnv()
}

// Environment variables for prompt version configuration.
const (
	EnvMemoVersion     = "MEMO_PROMPT_VERSION"
	EnvScheduleVersion = "SCHEDULE_PROMPT_VERSION"
	EnvAmazingVersion  = "AMAZING_PROMPT_VERSION"
)

// initFromEnv initializes prompt versions from environment variables.
// This allows runtime version selection without code changes.
func initFromEnv() {
	once.Do(func() {
		// Memo agent version
		if v := os.Getenv(EnvMemoVersion); v != "" {
			if version := PromptVersion(v); isValidPromptVersion(version) {
				_ = PromptRegistry.Memo.System.SetVersion(version) //nolint:errcheck // version set during init
			}
		}

		// Schedule agent version
		if v := os.Getenv(EnvScheduleVersion); v != "" {
			if version := PromptVersion(v); isValidPromptVersion(version) {
				_ = PromptRegistry.Schedule.System.SetVersion(version) //nolint:errcheck // version set during init
			}
		}

		// Amazing agent version
		if v := os.Getenv(EnvAmazingVersion); v != "" {
			if version := PromptVersion(v); isValidPromptVersion(version) {
				_ = PromptRegistry.Amazing.System.SetVersion(version) //nolint:errcheck // version set during init
				//nolint:errcheck // version set during init
				_ = PromptRegistry.Amazing.Planning.SetVersion(version)
				//nolint:errcheck // version set during init
				_ = PromptRegistry.Amazing.Synthesis.SetVersion(version)
			}
		}
	})
}

var once sync.Once

// isValidPromptVersion checks if a version is valid (has a registered template).
func isValidPromptVersion(version PromptVersion) bool {
	return version == PromptV1 || version == PromptV2
}

// GetMemoSystemPrompt returns the memo system prompt with variable substitution.
func GetMemoSystemPrompt(args ...any) string {
	return PromptRegistry.Memo.GetSystemPrompt(args...)
}

// GetScheduleSystemPrompt returns the schedule system prompt with timezone formatting.
// It handles the special case of 3 parameters: time, timezone, and tzOffset.
func GetScheduleSystemPrompt(time, timezone, tzOffset string) string {
	template := PromptRegistry.Schedule.System.GetTemplate()
	if template == "" {
		return ""
	}
	return fmt.Sprintf(template, time, timezone, tzOffset)
}

// GetAmazingPlanningPrompt returns the amazing planning prompt with variable substitution.
func GetAmazingPlanningPrompt(args ...any) string {
	return PromptRegistry.Amazing.GetPlanningPrompt(args...)
}

// GetAmazingSynthesisPrompt returns the amazing synthesis prompt with variable substitution.
func GetAmazingSynthesisPrompt(args ...any) string {
	return PromptRegistry.Amazing.GetSynthesisPrompt(args...)
}

// Exported for use in scheduler_v2.go.
func FormatTZOffset(offset int) string {
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

// SetPromptVersion sets the active prompt version for an agent type.
// Returns error if the version is not registered.
func SetPromptVersion(agentType string, version PromptVersion) error {
	PromptRegistry.mu.Lock()
	defer PromptRegistry.mu.Unlock()

	switch agentType {
	case "memo":
		return PromptRegistry.Memo.System.SetVersion(version)
	case "schedule":
		return PromptRegistry.Schedule.System.SetVersion(version)
	case "amazing":
		if err := PromptRegistry.Amazing.System.SetVersion(version); err != nil {
			return err
		}
		_ = PromptRegistry.Amazing.Planning.SetVersion(version) //nolint:errcheck //nolint:errcheck
		return PromptRegistry.Amazing.Synthesis.SetVersion(version)
	default:
		return fmt.Errorf("unknown agent type: %s", agentType)
	}
}

// GetPromptVersion returns the current active prompt version for an agent type.
// Thread-safe: uses read lock for concurrent access.
func GetPromptVersion(agentType string) PromptVersion {
	PromptRegistry.mu.RLock()
	defer PromptRegistry.mu.RUnlock()

	switch agentType {
	case "memo":
		return PromptRegistry.Memo.System.Version
	case "schedule":
		return PromptRegistry.Schedule.System.Version
	case "amazing":
		return PromptRegistry.Amazing.System.Version
	default:
		return PromptV1
	}
}

// ABConfig represents A/B testing configuration for a prompt experiment.
type ABConfig struct {
	ExperimentID     string
	ControlVersion   PromptVersion // V1 typically
	TreatmentVersion PromptVersion // V2 typically
	TrafficPercent   int           // 0-100, percentage for treatment
	Enabled          bool
}

// ABExperiment manages an A/B testing experiment for prompts.
type ABExperiment struct {
	config    ABConfig
	userIDMod int // Modulo for bucket assignment (default 100)
}

// NewABExperiment creates a new A/B experiment with the given configuration.
func NewABExperiment(config ABConfig) *ABExperiment {
	if config.TrafficPercent < 0 || config.TrafficPercent > 100 {
		config.TrafficPercent = 50 // Default to 50/50 split
	}
	userIDMod := 100 // Default modulo
	return &ABExperiment{
		config:    config,
		userIDMod: userIDMod,
	}
}

// GetVersionForUser returns the prompt version for a specific user based on A/B bucket.
// Users are deterministically assigned to buckets based on userID.
func (exp *ABExperiment) GetVersionForUser(userID int32) PromptVersion {
	if !exp.config.Enabled {
		return exp.config.ControlVersion
	}
	// Deterministic bucket assignment: userID % 100 < TrafficPercent → Treatment
	bucket := int(userID) % exp.userIDMod
	if bucket < exp.config.TrafficPercent {
		return exp.config.TreatmentVersion
	}
	return exp.config.ControlVersion
}

// Global experiments (can be configured at runtime).
var (
	MemoABExperiment     = NewABExperiment(ABConfig{ExperimentID: "memo-v1-v2", ControlVersion: PromptV1, TreatmentVersion: PromptV2, TrafficPercent: 0, Enabled: false})
	ScheduleABExperiment = NewABExperiment(ABConfig{ExperimentID: "schedule-v1-v2", ControlVersion: PromptV1, TreatmentVersion: PromptV2, TrafficPercent: 0, Enabled: false})
	AmazingABExperiment  = NewABExperiment(ABConfig{ExperimentID: "amazing-v1-v2", ControlVersion: PromptV1, TreatmentVersion: PromptV2, TrafficPercent: 0, Enabled: false})
)

// ConfigureABExperimentFromEnv configures A/B experiments from environment variables.
// Format: MEMO_AB_TRAFFIC=50 enables 50% traffic to V2.
func ConfigureABExperimentFromEnv() {
	if v := os.Getenv("MEMO_AB_TRAFFIC"); v != "" {
		if pct, err := strconv.Atoi(v); err == nil && pct > 0 && pct <= 100 {
			MemoABExperiment.config.TrafficPercent = pct
			MemoABExperiment.config.Enabled = true
		}
	}
	if v := os.Getenv("SCHEDULE_AB_TRAFFIC"); v != "" {
		if pct, err := strconv.Atoi(v); err == nil && pct > 0 && pct <= 100 {
			ScheduleABExperiment.config.TrafficPercent = pct
			ScheduleABExperiment.config.Enabled = true
		}
	}
	if v := os.Getenv("AMAZING_AB_TRAFFIC"); v != "" {
		if pct, err := strconv.Atoi(v); err == nil && pct > 0 && pct <= 100 {
			AmazingABExperiment.config.TrafficPercent = pct
			AmazingABExperiment.config.Enabled = true
		}
	}
}

// GetPromptVersionForUser returns the appropriate prompt version for a user,
// taking into account A/B experiments if enabled.
func GetPromptVersionForUser(agentType string, userID int32) PromptVersion {
	switch agentType {
	case "memo":
		return MemoABExperiment.GetVersionForUser(userID)
	case "schedule":
		return ScheduleABExperiment.GetVersionForUser(userID)
	case "amazing":
		return AmazingABExperiment.GetVersionForUser(userID)
	default:
		return PromptV1
	}
}

// MetricsRecorder defines the interface for recording prompt version metrics.
// This allows dependency injection for testing and different backends.
type MetricsRecorder interface {
	RecordPromptVersion(agentType, promptVersion string, success bool, latencyMs int64)
}

// Default metrics recorder (can be replaced with a real backend implementation).
var defaultMetricsRecorder MetricsRecorder = &noopMetricsRecorder{}

// SetMetricsRecorder sets the global metrics recorder.
func SetMetricsRecorder(recorder MetricsRecorder) {
	defaultMetricsRecorder = recorder
}

// noopMetricsRecorder is a no-op implementation used as default.
type noopMetricsRecorder struct{}

func (n *noopMetricsRecorder) RecordPromptVersion(agentType, promptVersion string, success bool, latencyMs int64) {
	// No-op by default
}

// RecordPromptUsage records a prompt usage with metrics.
// This should be called after each agent execution.
func RecordPromptUsage(agentType string, userID int32, success bool, latencyMs int64) {
	version := GetPromptVersionForUser(agentType, userID)
	if defaultMetricsRecorder != nil {
		defaultMetricsRecorder.RecordPromptVersion(agentType, string(version), success, latencyMs)
	}
}

// In-memory metrics for quick access (not persisted).
type promptMetricsSnapshot struct {
	requests   atomic.Int64
	successes  atomic.Int64
	latencySum atomic.Int64
}

var (
	memoMetricsV1     = &promptMetricsSnapshot{}
	memoMetricsV2     = &promptMetricsSnapshot{}
	scheduleMetricsV1 = &promptMetricsSnapshot{}
	scheduleMetricsV2 = &promptMetricsSnapshot{}
	amazingMetricsV1  = &promptMetricsSnapshot{}
	amazingMetricsV2  = &promptMetricsSnapshot{}
)

var (
	// metricsRegistry provides a lookup table for prompt version metrics.
	// This eliminates repetitive switch-case statements.
	// Protected by metricsRegistryMu for concurrent access.
	metricsRegistry = map[string]map[PromptVersion]*promptMetricsSnapshot{
		"memo": {
			PromptV1: memoMetricsV1,
			PromptV2: memoMetricsV2,
		},
		"schedule": {
			PromptV1: scheduleMetricsV1,
			PromptV2: scheduleMetricsV2,
		},
		"amazing": {
			PromptV1: amazingMetricsV1,
			PromptV2: amazingMetricsV2,
		},
	}
	metricsRegistryMu sync.RWMutex
)

// RecordPromptUsageInMemory records prompt usage to in-memory counters.
// This is a lightweight alternative for real-time monitoring.
// Concurrent-safe: uses RWMutex for map access, atomic operations for counters.
func RecordPromptUsageInMemory(agentType string, version PromptVersion, success bool, latencyMs int64) {
	metricsRegistryMu.RLock()
	versions, ok := metricsRegistry[agentType]
	metricsRegistryMu.RUnlock()

	if !ok {
		return
	}

	metricsRegistryMu.RLock()
	snapshot, ok := versions[version]
	metricsRegistryMu.RUnlock()

	if !ok {
		// Fall back to V1 if version not found
		metricsRegistryMu.RLock()
		snapshot = versions[PromptV1]
		metricsRegistryMu.RUnlock()
	}

	snapshot.requests.Add(1)
	if success {
		snapshot.successes.Add(1)
	}
	snapshot.latencySum.Add(latencyMs)
}

// GetPromptMetricsSnapshot returns the current in-memory metrics for a prompt version.
// Concurrent-safe: uses RWMutex for map access.
func GetPromptMetricsSnapshot(agentType string, version PromptVersion) (requests, successes int64, avgLatencyMs int64) {
	metricsRegistryMu.RLock()
	versions, ok := metricsRegistry[agentType]
	metricsRegistryMu.RUnlock()

	if !ok {
		return 0, 0, 0
	}

	metricsRegistryMu.RLock()
	snapshot, ok := versions[version]
	metricsRegistryMu.RUnlock()

	if !ok {
		metricsRegistryMu.RLock()
		snapshot = versions[PromptV1]
		metricsRegistryMu.RUnlock()
	}

	requests = snapshot.requests.Load()
	successes = snapshot.successes.Load()
	latencySum := snapshot.latencySum.Load()

	if requests > 0 {
		avgLatencyMs = latencySum / requests
	}

	return requests, successes, avgLatencyMs
}

// PromptExperimentReport represents a report of an A/B experiment's performance.
type PromptExperimentReport struct {
	GeneratedAt          time.Time
	AgentType            string
	ControlVersion       PromptVersion
	TreatmentVersion     PromptVersion
	Confidence           string
	Recommendation       string
	TreatmentRequests    int64
	ControlAvgLatency    int64
	ControlSuccessRate   float64
	TreatmentSuccesses   int64
	TreatmentSuccessRate float64
	TreatmentAvgLatency  int64
	SuccessRateDelta     float64
	LatencyDelta         int64
	ControlSuccesses     int64
	ControlRequests      int64
	TrafficPercent       int
}

// GenerateExperimentReport generates an A/B experiment report for an agent type.
func GenerateExperimentReport(agentType string) *PromptExperimentReport {
	var exp *ABExperiment
	var control, treatment PromptVersion

	switch agentType {
	case "memo":
		exp = MemoABExperiment
		control, treatment = PromptV1, PromptV2
	case "schedule":
		exp = ScheduleABExperiment
		control, treatment = PromptV1, PromptV2
	case "amazing":
		exp = AmazingABExperiment
		control, treatment = PromptV1, PromptV2
	default:
		return nil
	}

	controlReqs, controlSucc, controlLat := GetPromptMetricsSnapshot(agentType, control)
	treatmentReqs, treatmentSucc, treatmentLat := GetPromptMetricsSnapshot(agentType, treatment)

	report := &PromptExperimentReport{
		AgentType:        agentType,
		ControlVersion:   control,
		TreatmentVersion: treatment,
		TrafficPercent:   exp.config.TrafficPercent,

		ControlRequests:   controlReqs,
		ControlSuccesses:  controlSucc,
		ControlAvgLatency: controlLat,

		TreatmentRequests:   treatmentReqs,
		TreatmentSuccesses:  treatmentSucc,
		TreatmentAvgLatency: treatmentLat,

		GeneratedAt: time.Now(),
	}

	// Calculate rates
	if controlReqs > 0 {
		report.ControlSuccessRate = float64(controlSucc) / float64(controlReqs) * 100
	}
	if treatmentReqs > 0 {
		report.TreatmentSuccessRate = float64(treatmentSucc) / float64(treatmentReqs) * 100
	}

	// Calculate deltas
	report.SuccessRateDelta = report.TreatmentSuccessRate - report.ControlSuccessRate
	report.LatencyDelta = treatmentLat - controlLat

	// Determine recommendation
	report.Recommendation, report.Confidence = determineRecommendation(
		controlReqs, treatmentReqs,
		report.SuccessRateDelta, report.LatencyDelta,
	)

	return report
}

// determineRecommendation determines the experiment recommendation based on metrics.
func determineRecommendation(controlReqs, treatmentReqs int64, successDelta float64, latencyDelta int64) (recommendation, confidence string) {
	// Minimum sample size check
	minSamples := int64(100)
	if controlReqs < minSamples || treatmentReqs < minSamples {
		return "needs_more_data", "low"
	}

	// Success rate improvement is significant
	if successDelta >= 2.0 { // 2 percentage points improvement
		if latencyDelta <= 100 { // Latency not significantly worse
			return "rollout_treatment", "high"
		}
		return "rollout_treatment", "medium"
	}

	// Success rate degradation is significant
	if successDelta <= -2.0 {
		return "keep_control", "high"
	}

	// Within 2% - inconclusive
	if latencyDelta > 200 {
		return "keep_control", "medium" // Treatment is slower
	}

	return "needs_more_data", "medium"
}

// LogExperimentReport logs the experiment report to slog.
func LogExperimentReport(agentType string) {
	report := GenerateExperimentReport(agentType)
	if report == nil {
		slog.Warn("Failed to generate experiment report", "agent_type", agentType)
		return
	}

	slog.Info("A/B Experiment Report",
		"agent_type", report.AgentType,
		"control", report.ControlVersion,
		"treatment", report.TreatmentVersion,
		"traffic_percent", report.TrafficPercent,
		"control_requests", report.ControlRequests,
		"control_success_rate", fmt.Sprintf("%.2f%%", report.ControlSuccessRate),
		"treatment_requests", report.TreatmentRequests,
		"treatment_success_rate", fmt.Sprintf("%.2f%%", report.TreatmentSuccessRate),
		"success_delta", fmt.Sprintf("%.2fpp", report.SuccessRateDelta),
		"latency_delta", fmt.Sprintf("%dms", report.LatencyDelta),
		"recommendation", report.Recommendation,
		"confidence", report.Confidence,
	)
}
