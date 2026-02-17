package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hrygo/divinesense/ai/agents/universal"
	"github.com/hrygo/divinesense/ai/configloader"
)

// Config path for unified prompts
const (
	promptsConfigPath = "config/orchestrator/prompts.yaml"
)

// PromptConfig holds the orchestrator prompt templates.
type PromptConfig struct {
	Decomposer DecomposerPrompts
	Aggregator AggregatorPrompts
}

// DecomposerPrompts holds prompts for task decomposition.
type DecomposerPrompts struct {
	SystemContext        string `yaml:"system_context"`
	AnalysisInstructions string `yaml:"analysis_instructions"`
	OutputFormat         string `yaml:"output_format"`
	Rules                string `yaml:"rules"`
	UserRequestTemplate  string `yaml:"user_request_template"`
	TimeContextTemplate  string `yaml:"time_context_template"`
	Examples             string `yaml:"examples"`
}

// AggregatorPrompts holds prompts for result aggregation.
type AggregatorPrompts struct {
	SystemContext           string            `yaml:"system_context"`
	Requirements            string            `yaml:"requirements"`
	LanguageHints           map[string]string `yaml:"language_hints"`
	OriginalRequestTemplate string            `yaml:"original_request_template"`
	ResultsTemplate         string            `yaml:"results_template"`
	SynthesisStrategies     string            `yaml:"synthesis_strategies"`
}

// Global prompt config with lazy loading.
var (
	promptConfig     *PromptConfig
	promptConfigOnce sync.Once
	promptConfigErr  error
	configDir        string // Can be overridden for testing
)

// SetConfigDir overrides the default config directory.
func SetConfigDir(dir string) {
	configDir = dir
	promptConfigOnce = sync.Once{} // Reset for reload
	promptConfig = nil
}

// LoadPromptConfig loads the prompt configuration from YAML files.
func LoadPromptConfig() (*PromptConfig, error) {
	promptConfigOnce.Do(func() {
		loader := configloader.NewLoader(getBaseDir())
		var cfg PromptConfig
		err := loader.Load(promptsConfigPath, &cfg)
		if err != nil {
			promptConfigErr = fmt.Errorf("load prompts config: %w", err)
			return
		}

		// Override with custom configDir if set
		if configDir != "" {
			loader = configloader.NewLoader(configDir)
			if err := loader.Load("prompts.yaml", &cfg); err != nil {
				promptConfigErr = fmt.Errorf("load prompts config from custom dir: %w", err)
				return
			}
		}

		promptConfig = &cfg
	})

	return promptConfig, promptConfigErr
}

// getBaseDir returns the base directory for config files.
func getBaseDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(execPath)
}

// GetPromptConfig returns the global prompt config, loading if necessary.
// Falls back to defaults if config file fails to load.
func GetPromptConfig() *PromptConfig {
	cfg, err := LoadPromptConfig()
	if err != nil {
		return defaultPromptConfig()
	}
	return cfg
}

// defaultPromptConfig returns fallback prompts if config files fail to load.
func defaultPromptConfig() *PromptConfig {
	return &PromptConfig{
		Decomposer: DecomposerPrompts{
			SystemContext: "你是 DivineSense 的任务编排器。分析用户请求，将其分解为专家任务。",
			AnalysisInstructions: `## 分析步骤
1. **理解意图**: 用户想要什么？
2. **匹配专家**: 哪些专家适合处理？
3. **任务分解**: 为每个专家创建具体任务
4. **依赖分析**: 任务是否独立（并行）或有依赖（串行）`,
			OutputFormat: `## 输出格式 (仅 JSON，无 markdown)
{
  "analysis": "用户意图分析",
  "tasks": [
    {"agent": "专家名", "input": "具体输入", "purpose": "任务目的", "dependencies": ["依赖任务ID"]}
  ],
  "aggregate": true/false
}

### 字段说明
- dependencies: 依赖的任务ID列表。即前置任务，当前任务需等待其完成，并可使用其输出作为上下文。
- 无依赖: 空数组 [] 表示可立即并发执行。`,
			Rules: `## 规则
1. **默认并行**：只要任务间没有逻辑依赖，dependencies 设为空数组 []，系统会自动并行执行。
2. **按需串行**：只有当后置任务确实需要前置任务的数据时，才设置依赖。
3. **任务输入**：具体、可执行，包含必要的上下文。`,
			UserRequestTemplate: "## 用户请求\n%s",
			TimeContextTemplate: `## Current Time Context
%s

**Important**: Use the above time context to resolve relative dates (e.g., "明天" = %s, "下周三" = calculate from this week).`,
			Examples: `## Examples
User: "查一下上海的天气，并发邮件给老板"
Output:
{
  "analysis": "天气查询(Task 1) -> 邮件发送(Task 2, 依赖Task 1结果)",
  "tasks": [
    {"id": "t1", "agent": "weather", "input": "上海天气", "purpose": "获取天气信息", "dependencies": []},
    {"id": "t2", "agent": "email", "input": "基于前置任务(t1)的天气结果，发邮件", "purpose": "发送信息", "dependencies": ["t1"]}
  ],
  "aggregate": true
}`,
		},
		Aggregator: AggregatorPrompts{
			SystemContext: "你是 DivineSense 的结果整合助手。将多个专家的结果合并为连贯回复。",
			Requirements: `## 整合要求
1. 完整回答用户请求，不遗漏信息
2. 自然融合多源信息，避免重复
3. 突出最相关的信息
4. 保持友好、专业的语气
5. 使用清晰的格式（列表、分段等）`,
			LanguageHints: map[string]string{
				"zh":      "中文（与用户语言一致）",
				"en":      "English (matching user's language)",
				"default": "中文（与用户语言一致）",
			},
			OriginalRequestTemplate: "## 用户原始请求\n%s",
			ResultsTemplate:         "## 专家结果\n%s",
			SynthesisStrategies: `## 整合策略
- 日程+笔记：先展示日程(表格)，再展示笔记(引用)。
- 多笔记搜索：按相关度分组展示。
- 冲突处理：明确指出冲突并提供建议。`,
		},
	}
}

// BuildDecomposerPrompt builds the full decomposition prompt from config.
// If history is provided, it will be included to enable context-aware task decomposition.
func (c *PromptConfig) BuildDecomposerPrompt(userInput, expertDescriptions string, timeContext *universal.TimeContext, history []string) string {
	d := c.Decomposer

	// Build time context section
	timeContextSection := ""
	if timeContext != nil && d.TimeContextTemplate != "" {
		timeContextSection = fmt.Sprintf(d.TimeContextTemplate,
			timeContext.FormatAsJSONBlock(),
			timeContext.Relative.Tomorrow)
	}

	// Build conversation history section (for context-aware decomposition)
	historySection := ""
	if len(history) > 0 {
		historySection = "## Conversation History\n以上是对话历史，请结合历史上下文理解用户意图。\n\n"
		// Format history as user/assistant pairs
		for i := 0; i < len(history); i += 2 {
			if i+1 < len(history) {
				historySection += fmt.Sprintf("**User**: %s\n\n**Assistant**: %s\n\n", history[i], history[i+1])
			} else {
				historySection += fmt.Sprintf("**User**: %s\n\n", history[i])
			}
		}
	}

	return fmt.Sprintf(`%s

## Available Expert Agents
%s

%s
%s
%s
%s
%s
%s

%s`, d.SystemContext, expertDescriptions, historySection, timeContextSection, d.AnalysisInstructions, d.OutputFormat, d.Rules, d.Examples,
		fmt.Sprintf(d.UserRequestTemplate, userInput))
}

// BuildAggregatorPrompt builds the full aggregation prompt from config.
func (c *PromptConfig) BuildAggregatorPrompt(analysis string, results []string, userLang string) string {
	a := c.Aggregator
	langHint := a.LanguageHints["default"]
	if hint, ok := a.LanguageHints[userLang]; ok {
		langHint = hint
	}

	// Use strings.Builder for efficient concatenation
	var resultsBuilder strings.Builder
	for i, r := range results {
		resultsBuilder.WriteString(fmt.Sprintf("【专家 %d】\n%s\n\n", i+1, r))
	}
	resultsStr := resultsBuilder.String()

	return fmt.Sprintf(`%s

%s
%s

%s

## 输出语言
%s`, a.SystemContext,
		fmt.Sprintf(a.OriginalRequestTemplate, analysis),
		fmt.Sprintf(a.ResultsTemplate, resultsStr),
		a.Requirements,
		langHint)
}
