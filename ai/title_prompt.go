package ai

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/hrygo/divinesense/ai/configloader"
)

// TitlePromptConfig holds the configuration for title generation.
type TitlePromptConfig struct {
	Name                 string `yaml:"name"`
	Version              string `yaml:"version"`
	SystemPrompt         string `yaml:"system_prompt"`
	ConversationTemplate string `yaml:"conversation_template"`
	MemoTemplate         string `yaml:"memo_template"`
	Params               struct {
		MaxTokens          int     `yaml:"max_tokens"`
		Temperature        float64 `yaml:"temperature"`
		TimeoutSeconds     int     `yaml:"timeout_seconds"`
		InputTruncateChars int     `yaml:"input_truncate_chars"`
		MaxRunes           int     `yaml:"max_runes"`
	} `yaml:"params"`
}

// ConversationPromptData holds data for conversation title template.
type ConversationPromptData struct {
	UserMessage string
	AIResponse  string
}

// MemoPromptData holds data for memo title template.
type MemoPromptData struct {
	Content string
	Title   string
}

// Global config with lazy loading
var (
	titleConfig     *TitlePromptConfig
	titleConfigOnce sync.Once
	titleConfigErr  error
	titleConfigDir  string // Can be overridden for testing
)

// SetTitleConfigDir overrides the default config directory.
func SetTitleConfigDir(dir string) {
	titleConfigDir = dir
	titleConfigOnce = sync.Once{} // Reset for reload
	titleConfig = nil
}

// LoadTitlePromptConfig loads the title prompt configuration from YAML.
func LoadTitlePromptConfig() (*TitlePromptConfig, error) {
	titleConfigOnce.Do(func() {
		loader := configloader.NewLoader(getTitleConfigBaseDir())
		var cfg TitlePromptConfig
		err := loader.Load("config/prompts/title.yaml", &cfg)
		if err != nil {
			titleConfigErr = fmt.Errorf("load title prompts config: %w", err)
			return
		}

		// Override with custom titleConfigDir if set
		if titleConfigDir != "" {
			loader = configloader.NewLoader(titleConfigDir)
			if err := loader.Load("title.yaml", &cfg); err != nil {
				titleConfigErr = fmt.Errorf("load title config from custom dir: %w", err)
				return
			}
		}

		titleConfig = &cfg
	})

	return titleConfig, titleConfigErr
}

// getTitleConfigBaseDir returns the base directory for config files.
func getTitleConfigBaseDir() string {
	if titleConfigDir != "" {
		return titleConfigDir
	}
	execPath, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(execPath)
}

// GetTitlePromptConfig returns the global title prompt config, loading if necessary.
// Falls back to defaults if config file fails to load.
func GetTitlePromptConfig() *TitlePromptConfig {
	cfg, err := LoadTitlePromptConfig()
	if err != nil {
		return defaultTitlePromptConfig()
	}
	return cfg
}

// defaultTitlePromptConfig returns fallback prompts if config files fail to load.
func defaultTitlePromptConfig() *TitlePromptConfig {
	return &TitlePromptConfig{
		Name:    "title",
		Version: "default",
		SystemPrompt: `你是一个专业的对话标题生成助手。你的任务是根据用户和AI的对话内容，生成一个简洁、准确的标题。

要求：
1. 标题长度：3-15个字符（中文）或 3-8个单词（英文）
2. 标题应该反映对话的核心主题
3. 使用简洁的语言，避免使用"关于..."、"讨论了..."等冗余表述
4. 如果是问题，可以直接用问题本身作为标题
5. 如果是任务，可以用任务描述作为标题
6. 保持中立客观的语气

请直接返回JSON格式：{"title": "生成的标题"}`,
		ConversationTemplate: `用户消息: {{.UserMessage}}

AI 回复: {{.AIResponse}}

请为这段对话生成一个简短的标题。`,
		MemoTemplate: `笔记标题: {{.Title}}

笔记内容:
{{.Content}}

请为以下笔记生成一个简短的标题。`,
		Params: struct {
			MaxTokens          int     `yaml:"max_tokens"`
			Temperature        float64 `yaml:"temperature"`
			TimeoutSeconds     int     `yaml:"timeout_seconds"`
			InputTruncateChars int     `yaml:"input_truncate_chars"`
			MaxRunes           int     `yaml:"max_runes"`
		}{
			MaxTokens:          50,
			Temperature:        0.1,
			TimeoutSeconds:     30,
			InputTruncateChars: 500,
			MaxRunes:           50,
		},
	}
}

// BuildConversationPrompt builds the user prompt for conversation title generation.
func (c *TitlePromptConfig) BuildConversationPrompt(data *ConversationPromptData) (string, error) {
	tmpl, err := template.New("conversation").Parse(c.ConversationTemplate)
	if err != nil {
		return "", fmt.Errorf("parse conversation template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute conversation template: %w", err)
	}

	return buf.String(), nil
}

// BuildMemoPrompt builds the user prompt for memo title generation.
func (c *TitlePromptConfig) BuildMemoPrompt(data *MemoPromptData) (string, error) {
	tmpl, err := template.New("memo").Parse(c.MemoTemplate)
	if err != nil {
		return "", fmt.Errorf("parse memo template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute memo template: %w", err)
	}

	return buf.String(), nil
}
