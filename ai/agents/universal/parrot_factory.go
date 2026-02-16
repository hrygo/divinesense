// Package universal provides the factory for creating configuration-driven parrots.
package universal

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/hrygo/divinesense/ai"
	"github.com/hrygo/divinesense/ai/agents"
	"github.com/hrygo/divinesense/ai/agents/registry"
)

// ParrotFactory creates parrots from configuration.
// It provides both generic CreateParrot and specific helpers
// (CreateMemoParrot, CreateScheduleParrot, etc.).
type ParrotFactory struct {
	mu               sync.RWMutex
	configs          map[string]*ParrotConfig
	configDir        string
	llm              ai.LLMService
	baseURL          string                     // Frontend base URL for generating links
	toolFactories    map[string]ToolFactoryFunc // Dynamic tool creation
	retrieverFactory func() any                 // Retriever factory
	scheduleFactory  func() any                 // Schedule service factory
}

// ToolFactoryFunc creates a tool with given userID.
type ToolFactoryFunc func(userID int32) (agent.ToolWithSchema, error)

// FactoryOption configures the ParrotFactory.
type FactoryOption func(*ParrotFactory) error

// WithConfigDir sets the configuration directory.
func WithConfigDir(dir string) FactoryOption {
	return func(f *ParrotFactory) error {
		if dir == "" {
			return errors.New("config directory cannot be empty")
		}
		// Verify directory exists
		if info, err := os.Stat(dir); err != nil {
			return fmt.Errorf("config directory not accessible: %w", err)
		} else if !info.IsDir() {
			return fmt.Errorf("config path is not a directory: %s", dir)
		}
		f.configDir = dir
		return nil
	}
}

// WithLLM sets the LLM service.
func WithLLM(llm ai.LLMService) FactoryOption {
	return func(f *ParrotFactory) error {
		if llm == nil {
			return errors.New("LLM service cannot be nil")
		}
		f.llm = llm
		return nil
	}
}

// WithToolFactories sets the tool factory functions.
func WithToolFactories(factories map[string]ToolFactoryFunc) FactoryOption {
	return func(f *ParrotFactory) error {
		f.toolFactories = factories
		return nil
	}
}

// WithGlobalToolFactories loads tool factories from the global registry.
// This is a convenience option that automatically imports all registered factories.
func WithGlobalToolFactories() FactoryOption {
	return func(f *ParrotFactory) error {
		// Import from global registry
		f.toolFactories = convertRegistryFactories(registry.BuildToolFactoriesMap())
		return nil
	}
}

// WithMergedToolFactories merges the provided factories with global registry factories.
// Provided factories take precedence over global registry.
func WithMergedToolFactories(factories map[string]ToolFactoryFunc) FactoryOption {
	return func(f *ParrotFactory) error {
		// Start with global registry
		f.toolFactories = convertRegistryFactories(registry.BuildToolFactoriesMap())
		// Merge with provided factories (overwriting conflicts)
		for name, factory := range factories {
			f.toolFactories[name] = factory
		}
		return nil
	}
}

// convertRegistryFactories converts registry.ToolFactoryFunc to universal.ToolFactoryFunc.
// Both types have identical signatures but are defined in different packages.
func convertRegistryFactories(input map[string]registry.ToolFactoryFunc) map[string]ToolFactoryFunc {
	result := make(map[string]ToolFactoryFunc, len(input))
	for name, factory := range input {
		// Direct type conversion since both have identical signatures
		result[name] = ToolFactoryFunc(factory)
	}
	return result
}

// WithRetriever sets the retriever factory function.
func WithRetriever(retrieverFactory func() any) FactoryOption {
	return func(f *ParrotFactory) error {
		f.retrieverFactory = retrieverFactory
		return nil
	}
}

// WithScheduleService sets the schedule service factory function.
func WithScheduleService(scheduleFactory func() any) FactoryOption {
	return func(f *ParrotFactory) error {
		f.scheduleFactory = scheduleFactory
		return nil
	}
}

// WithBaseURL sets the frontend base URL for generating links in prompts.
func WithBaseURL(baseURL string) FactoryOption {
	return func(f *ParrotFactory) error {
		f.baseURL = baseURL
		return nil
	}
}

// NewParrotFactory creates a new ParrotFactory with options.
func NewParrotFactory(opts ...FactoryOption) (*ParrotFactory, error) {
	factory := &ParrotFactory{
		configs:       make(map[string]*ParrotConfig),
		toolFactories: make(map[string]ToolFactoryFunc),
	}

	// Set default config dir, but allow environment variable override
	configDir := "./config/parrots"
	if envDir := os.Getenv("DIVINESENSE_PARROT_CONFIG_DIR"); envDir != "" {
		configDir = envDir
	}
	factory.configDir = configDir

	for _, opt := range opts {
		if err := opt(factory); err != nil {
			return nil, err
		}
	}

	// Load configurations from directory (ignore errors if dir doesn't exist)
	if err := factory.LoadConfigs(); err != nil {
		// Log but don't fail - configs can be registered programmatically
		slog.Warn("Failed to load parrot configs from directory", "dir", configDir, "error", err)
	}

	return factory, nil
}

// LoadConfigs loads all parrot configurations from the config directory.
func (f *ParrotFactory) LoadConfigs() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := os.ReadDir(f.configDir)
	if err != nil {
		return fmt.Errorf("read config dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Load YAML files
		if filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml" {
			configPath := filepath.Join(f.configDir, entry.Name())
			config, err := LoadParrotConfig(configPath)
			if err != nil {
				return fmt.Errorf("load config %s: %w", entry.Name(), err)
			}

			f.configs[config.Name] = config
		}
	}

	return nil
}

// RegisterConfig registers a parrot configuration programmatically.
func (f *ParrotFactory) RegisterConfig(config *ParrotConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if config.Name == "" {
		return errors.New("config name cannot be empty")
	}

	f.configs[config.Name] = config
	return nil
}

// GetConfig retrieves a parrot configuration by name.
func (f *ParrotFactory) GetConfig(name string) (*ParrotConfig, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	config, ok := f.configs[name]
	return config, ok
}

// ListConfigs returns all registered parrot names.
func (f *ParrotFactory) ListConfigs() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.configs))
	for name := range f.configs {
		names = append(names, name)
	}
	return names
}

// CreateParrot creates a parrot from configuration by name.
func (f *ParrotFactory) CreateParrot(name string, userID int32) (agent.ParrotAgent, error) {
	config, ok := f.GetConfig(name)
	if !ok {
		return nil, fmt.Errorf("parrot config not found: %s", name)
	}

	return f.CreateParrotFromConfig(config, userID)
}

// CreateParrotFromConfig creates a parrot from a configuration.
func (f *ParrotFactory) CreateParrotFromConfig(config *ParrotConfig, userID int32) (agent.ParrotAgent, error) {
	// Inject baseURL from factory if not already set in config
	if config.BaseURL == "" && f.baseURL != "" {
		config.BaseURL = f.baseURL
	}

	// Resolve tools from factory functions
	tools := make(map[string]agent.ToolWithSchema)
	for _, toolName := range config.Tools {
		toolFactory, ok := f.toolFactories[toolName]
		if !ok {
			return nil, fmt.Errorf("tool factory not found: %s", toolName)
		}
		tool, err := toolFactory(userID)
		if err != nil {
			return nil, fmt.Errorf("create tool %s: %w", toolName, err)
		}
		tools[toolName] = tool
	}

	// Create UniversalParrot
	parrot, err := NewUniversalParrot(config, f.llm, tools, userID)
	if err != nil {
		return nil, fmt.Errorf("create universal parrot: %w", err)
	}

	return parrot, nil
}

// CreateMemoParrot creates a UniversalParrot configured as memo parrot.
func (f *ParrotFactory) CreateMemoParrot(userID int32, retriever any) (agent.ParrotAgent, error) {
	config, ok := f.GetConfig("memo")
	if !ok {
		// Fallback to default config
		config = DefaultMemoParrotConfig()
	}

	parrot, err := f.CreateParrotFromConfig(config, userID)
	if err != nil {
		return nil, err
	}

	// Type assert to UniversalParrot to set retriever
	if up, ok := parrot.(*UniversalParrot); ok && retriever != nil {
		up.SetRetriever(retriever)
	}

	return parrot, nil
}

// CreateScheduleParrot creates a UniversalParrot configured as schedule parrot.
func (f *ParrotFactory) CreateScheduleParrot(userID int32, scheduleService any) (agent.ParrotAgent, error) {
	config, ok := f.GetConfig("schedule")
	if !ok {
		// Fallback to default config
		config = DefaultScheduleParrotConfig()
	}

	parrot, err := f.CreateParrotFromConfig(config, userID)
	if err != nil {
		return nil, err
	}

	// Type assert to UniversalParrot to set schedule service
	if up, ok := parrot.(*UniversalParrot); ok && scheduleService != nil {
		up.SetScheduleService(scheduleService)
	}

	return parrot, nil
}
