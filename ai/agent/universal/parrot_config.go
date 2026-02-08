// Package universal provides configuration structures for UniversalParrot.
package universal

import (
	"time"

	"github.com/hrygo/divinesense/ai/agent"
)

// ParrotConfig defines a parrot's behavior declaratively.
// This configuration-driven approach allows creating new parrots
// without writing code, just by defining a YAML config file.
type ParrotConfig struct {
	// Identity
	Name        string `json:"name" yaml:"name"`
	DisplayName string `json:"display_name" yaml:"display_name"`
	Emoji       string `json:"emoji" yaml:"emoji"`

	// Execution
	Strategy      StrategyType `json:"strategy" yaml:"strategy"`
	MaxIterations int          `json:"max_iterations" yaml:"max_iterations"`

	// Capabilities
	Tools []string `json:"tools" yaml:"tools"`

	// Prompts
	SystemPrompt string   `json:"system_prompt" yaml:"system_prompt"`
	PromptHints  []string `json:"prompt_hints" yaml:"prompt_hints"`

	// Behavior
	EnableCache bool          `json:"enable_cache" yaml:"enable_cache"`
	CacheTTL    time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
	CacheSize   int           `json:"cache_size" yaml:"cache_size"`

	// Metadata
	SelfDescription *agent.ParrotSelfCognition `json:"self_description" yaml:"self_description"`
}

// ToolSetConfig defines a set of tools for a parrot.
// This is used by ToolSetRegistry to register tool sets.
type ToolSetConfig struct {
	// Name is the unique identifier for this tool set.
	Name string `json:"name"`

	// AgentType is the type of parrot this tool set belongs to.
	AgentType string `json:"agent_type"`

	// SystemPrompt is the prompt for this parrot.
	SystemPrompt string `json:"system_prompt"`

	// Tools is the list of tool descriptors.
	Tools []ToolDescriptor `json:"tools"`

	// Execution mode
	NativeCalling bool `json:"native_calling"`

	// FastPath configuration (optional)
	FastPath *FastPathConfig `json:"fast_path,omitempty"`
}

// FastPathConfig defines optimization for simple CRUD operations.
type FastPathConfig struct {
	// DirectAnswerPatterns are patterns for simple queries that don't need tools.
	DirectAnswerPatterns []string `json:"direct_answer_patterns"`

	// SimpleSchedulePatterns are patterns for simple schedule additions.
	SimpleSchedulePatterns []string `json:"simple_schedule_patterns"`

	// SimpleMemoPatterns are patterns for simple memo searches.
	SimpleMemoPatterns []string `json:"simple_memo_patterns"`

	// Enabled turns on fast path optimization.
	Enabled bool `json:"enabled"`
}

// ToolDescriptor describes a tool that can be used by parrots.
type ToolDescriptor struct {
	// Name is the unique identifier for this tool.
	Name string `json:"name"`

	// Description describes what this tool does.
	Description string `json:"description"`

	// Parameters is the JSON Schema for the tool's input.
	Parameters map[string]interface{} `json:"parameters"`

	// Factory creates a new instance of this tool.
	Factory func() (agent.ToolWithSchema, error) `json:"-"`
}
