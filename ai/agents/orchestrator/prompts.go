package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hrygo/divinesense/ai/agents/universal"
	"gopkg.in/yaml.v3"
)

// Default config paths relative to project root
const (
	decomposerPromptsPath = "config/orchestrator/decomposer.yaml"
	aggregatorPromptsPath = "config/orchestrator/aggregator.yaml"
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
}

// AggregatorPrompts holds prompts for result aggregation.
type AggregatorPrompts struct {
	SystemContext           string            `yaml:"system_context"`
	Requirements            string            `yaml:"requirements"`
	LanguageHints           map[string]string `yaml:"language_hints"`
	OriginalRequestTemplate string            `yaml:"original_request_template"`
	ResultsTemplate         string            `yaml:"results_template"`
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
		decomposerPath := decomposerPromptsPath
		aggregatorPath := aggregatorPromptsPath

		if configDir != "" {
			decomposerPath = filepath.Join(configDir, "decomposer.yaml")
			aggregatorPath = filepath.Join(configDir, "aggregator.yaml")
		}

		decomposer, err := loadDecomposerPrompts(decomposerPath)
		if err != nil {
			promptConfigErr = fmt.Errorf("load decomposer prompts: %w", err)
			return
		}

		aggregator, err := loadAggregatorPrompts(aggregatorPath)
		if err != nil {
			promptConfigErr = fmt.Errorf("load aggregator prompts: %w", err)
			return
		}

		promptConfig = &PromptConfig{
			Decomposer: decomposer,
			Aggregator: aggregator,
		}
	})

	return promptConfig, promptConfigErr
}

// loadDecomposerPrompts loads decomposer prompts from YAML file.
func loadDecomposerPrompts(path string) (DecomposerPrompts, error) {
	data, err := readFileWithFallback(path)
	if err != nil {
		return DecomposerPrompts{}, err
	}

	var prompts DecomposerPrompts
	if err := yaml.Unmarshal(data, &prompts); err != nil {
		return DecomposerPrompts{}, fmt.Errorf("parse YAML: %w", err)
	}

	return prompts, nil
}

// loadAggregatorPrompts loads aggregator prompts from YAML file.
func loadAggregatorPrompts(path string) (AggregatorPrompts, error) {
	data, err := readFileWithFallback(path)
	if err != nil {
		return AggregatorPrompts{}, err
	}

	var prompts AggregatorPrompts
	if err := yaml.Unmarshal(data, &prompts); err != nil {
		return AggregatorPrompts{}, fmt.Errorf("parse YAML: %w", err)
	}

	return prompts, nil
}

// readFileWithFallback tries to read file from path, then from executable directory.
func readFileWithFallback(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, nil
	}

	// Try relative to executable for production builds
	execPath, _ := os.Executable()
	absPath := filepath.Join(filepath.Dir(execPath), path)
	return os.ReadFile(absPath)
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
    {"agent": "专家名", "input": "具体输入", "purpose": "任务目的"}
  ],
  "parallel": true/false,
  "aggregate": true/false
}`,
			Rules: `## 规则
- 仅使用列出的专家
- 单专家时 parallel=false, aggregate=false
- 多专家独立时 parallel=true, aggregate=true
- 任务输入要具体、可执行`,
			UserRequestTemplate: "## 用户请求\n%s",
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
		},
	}
}

// BuildDecomposerPrompt builds the full decomposition prompt from config.
func (c *PromptConfig) BuildDecomposerPrompt(userInput, expertDescriptions string, timeContext *universal.TimeContext) string {
	d := c.Decomposer

	// Build time context section
	timeContextSection := ""
	if timeContext != nil {
		timeContextSection = fmt.Sprintf(`## Current Time Context
%s

**Important**: Use the above time context to resolve relative dates (e.g., "明天" = %s, "下周三" = calculate from this week).
`, timeContext.FormatAsJSONBlock(), timeContext.Relative.Tomorrow)
	}

	return fmt.Sprintf(`%s

## Available Expert Agents
%s

%s
%s
%s
%s

%s`, d.SystemContext, expertDescriptions, timeContextSection, d.AnalysisInstructions, d.OutputFormat, d.Rules,
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
