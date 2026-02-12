// Package universal provides configuration loading for UniversalParrot.
package universal

import (
	"fmt"
	"os"
	"time"

	"github.com/hrygo/divinesense/ai/agents"
	"gopkg.in/yaml.v3"
)

// LoadParrotConfig loads a parrot configuration from a YAML file.
func LoadParrotConfig(path string) (*ParrotConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var config ParrotConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	// Set defaults
	if config.MaxIterations <= 0 {
		config.MaxIterations = 10
	}
	if config.Strategy == "" {
		config.Strategy = StrategyReAct
	}
	if config.EnableCache && config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}
	if config.CacheSize <= 0 {
		config.CacheSize = 100
	}

	return &config, nil
}

// SaveParrotConfig saves a parrot configuration to a YAML file.
func SaveParrotConfig(config *ParrotConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// DefaultMemoParrotConfig returns the default configuration for MemoParrot.
func DefaultMemoParrotConfig() *ParrotConfig {
	return &ParrotConfig{
		Name:        "memo",
		DisplayName: "Memo Parrot",
		Emoji:       "📝",
		Strategy:    StrategyReAct,
		Tools:       []string{"memo_search"},
		SystemPrompt: `You are a helpful assistant for searching and retrieving notes.
You can search through the user's notes using semantic search.

When the user asks about their notes or wants to find information:
1. Use the memo_search tool with relevant keywords
2. Present the results in a clear, organized way
3. If no relevant notes are found, suggest alternative search terms

Be concise and helpful in your responses.`,
		PromptHints: []string{
			"搜索我的笔记",
			"找一下关于",
			"查看我的记录",
		},
		MaxIterations: 10,
		EnableCache:   true,
		CacheSize:     100,
		CacheTTL:      5 * time.Minute,
		SelfDescription: &agent.ParrotSelfCognition{
			Title:        "Memo Parrot",
			Name:         "memo",
			Emoji:        "📝",
			Capabilities: []string{"memo_search"},
		},
	}
}

// DefaultScheduleParrotConfig returns the default configuration for ScheduleParrot.
func DefaultScheduleParrotConfig() *ParrotConfig {
	return &ParrotConfig{
		Name:        "schedule",
		DisplayName: "Schedule Parrot",
		Emoji:       "📅",
		Strategy:    StrategyDirect,
		Tools:       []string{"schedule_add", "schedule_query", "schedule_update", "find_free_time"},
		SystemPrompt: `You are a helpful assistant for managing schedules and calendars.

You can help users:
- Create new schedule entries
- Query existing schedules
- Update existing schedule entries
- Find free time slots

When creating schedules:
- Always use ISO 8601 format for times (e.g., 2026-02-09T10:00:00+08:00)
- Detect conflicts and warn the user
- Confirm the schedule details after creation

Be concise and helpful in your responses.`,
		PromptHints: []string{
			"帮我安排",
			"新建日程",
			"查看今天的安排",
		},
		MaxIterations: 5,
		EnableCache:   true,
		CacheSize:     50,
		CacheTTL:      5 * time.Minute,
		SelfDescription: &agent.ParrotSelfCognition{
			Title:        "Schedule Parrot",
			Name:         "schedule",
			Emoji:        "📅",
			Capabilities: []string{"schedule_add", "schedule_query", "schedule_update", "find_free_time"},
		},
	}
}
