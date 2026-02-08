// Package registry provides tool factory registration for dynamic tool creation.
package registry

import (
	"fmt"
	"sync"

	"github.com/hrygo/divinesense/ai/agent"
)

// ToolFactoryFunc creates a tool with given userID.
// This matches the signature used in universal/parrot_factory.go.
type ToolFactoryFunc func(userID int32) (agent.ToolWithSchema, error)

// ToolFactoryEntry wraps a factory function with metadata.
type ToolFactoryEntry struct {
	Factory  ToolFactoryFunc
	Metadata ToolMetadata
}

// ToolFactoryRegistry manages tool factory registration.
// This allows dynamic tool creation based on user context.
type ToolFactoryRegistry struct {
	mu        sync.RWMutex
	factories map[string]*ToolFactoryEntry
}

// Global tool factory registry instance.
var globalFactoryRegistry = NewToolFactoryRegistry()

// NewToolFactoryRegistry creates a new tool factory registry.
func NewToolFactoryRegistry() *ToolFactoryRegistry {
	return &ToolFactoryRegistry{
		factories: make(map[string]*ToolFactoryEntry),
	}
}

// RegisterFactory registers a tool factory globally.
func RegisterFactory(name string, factory ToolFactoryFunc) error {
	return RegisterFactoryWithMetadata(name, factory, ToolMetadata{})
}

// RegisterFactoryWithMetadata registers a tool factory with metadata.
func RegisterFactoryWithMetadata(name string, factory ToolFactoryFunc, metadata ToolMetadata) error {
	globalFactoryRegistry.mu.Lock()
	defer globalFactoryRegistry.mu.Unlock()

	if _, exists := globalFactoryRegistry.factories[name]; exists {
		return fmt.Errorf("tool factory already registered: %s", name)
	}

	// Set default category if not specified
	if metadata.Category == "" {
		metadata.Category = inferCategory(name)
	}

	globalFactoryRegistry.factories[name] = &ToolFactoryEntry{
		Factory:  factory,
		Metadata: metadata,
	}
	return nil
}

// RegisterFactoryInCategory registers a tool factory in a specific category.
func RegisterFactoryInCategory(category ToolCategory, name string, factory ToolFactoryFunc) error {
	return RegisterFactoryWithMetadata(name, factory, ToolMetadata{Category: category})
}

// GetFactory retrieves a tool factory by name.
func GetFactory(name string) (ToolFactoryFunc, bool) {
	globalFactoryRegistry.mu.RLock()
	defer globalFactoryRegistry.mu.RUnlock()

	entry, ok := globalFactoryRegistry.factories[name]
	if !ok {
		return nil, false
	}
	return entry.Factory, true
}

// MustGetFactory retrieves a tool factory by name or panics.
func MustGetFactory(name string) ToolFactoryFunc {
	factory, ok := GetFactory(name)
	if !ok {
		panic(fmt.Sprintf("tool factory not found: %s", name))
	}
	return factory
}

// CreateTool creates a tool using a registered factory.
func CreateTool(name string, userID int32) (agent.ToolWithSchema, error) {
	factory, ok := GetFactory(name)
	if !ok {
		return nil, fmt.Errorf("tool factory not found: %s", name)
	}
	return factory(userID)
}

// ListFactories returns all registered factory names.
func ListFactories() []string {
	globalFactoryRegistry.mu.RLock()
	defer globalFactoryRegistry.mu.RUnlock()

	names := make([]string, 0, len(globalFactoryRegistry.factories))
	for name := range globalFactoryRegistry.factories {
		names = append(names, name)
	}
	return names
}

// ListFactoriesByCategory returns all factory names in a specific category.
func ListFactoriesByCategory(category ToolCategory) []string {
	globalFactoryRegistry.mu.RLock()
	defer globalFactoryRegistry.mu.RUnlock()

	names := make([]string, 0)
	for name, entry := range globalFactoryRegistry.factories {
		if entry.Metadata.Category == category {
			names = append(names, name)
		}
	}
	return names
}

// GetFactoryMetadata retrieves metadata for a factory.
func GetFactoryMetadata(name string) (ToolMetadata, bool) {
	globalFactoryRegistry.mu.RLock()
	defer globalFactoryRegistry.mu.RUnlock()

	entry, ok := globalFactoryRegistry.factories[name]
	if !ok {
		return ToolMetadata{}, false
	}
	return entry.Metadata, true
}

// UnregisterFactory removes a factory from the registry.
// This is primarily used for testing.
func UnregisterFactory(name string) {
	globalFactoryRegistry.mu.Lock()
	defer globalFactoryRegistry.mu.Unlock()
	delete(globalFactoryRegistry.factories, name)
}

// ClearFactories removes all factories from the registry.
// This is primarily used for testing.
func ClearFactories() {
	globalFactoryRegistry.mu.Lock()
	defer globalFactoryRegistry.mu.Unlock()
	globalFactoryRegistry.factories = make(map[string]*ToolFactoryEntry)
}

// FactoryCount returns the number of registered factories.
func FactoryCount() int {
	globalFactoryRegistry.mu.RLock()
	defer globalFactoryRegistry.mu.RUnlock()
	return len(globalFactoryRegistry.factories)
}

// GetAllFactories returns all factories as a map for easy iteration.
func GetAllFactories() map[string]ToolFactoryFunc {
	globalFactoryRegistry.mu.RLock()
	defer globalFactoryRegistry.mu.RUnlock()

	result := make(map[string]ToolFactoryFunc, len(globalFactoryRegistry.factories))
	for name, entry := range globalFactoryRegistry.factories {
		result[name] = entry.Factory
	}
	return result
}

// GetAllFactoryEntries returns all factory entries with metadata.
func GetAllFactoryEntries() map[string]*ToolFactoryEntry {
	globalFactoryRegistry.mu.RLock()
	defer globalFactoryRegistry.mu.RUnlock()

	result := make(map[string]*ToolFactoryEntry, len(globalFactoryRegistry.factories))
	for name, entry := range globalFactoryRegistry.factories {
		result[name] = entry
	}
	return result
}

// BuildToolFactoriesMap constructs a map suitable for passing to ParrotFactory.
// This integrates the global factory registry with ParrotFactory's WithToolFactories option.
func BuildToolFactoriesMap() map[string]ToolFactoryFunc {
	return GetAllFactories()
}
