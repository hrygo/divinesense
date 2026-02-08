// Package registry provides tool and prompt registration for UniversalParrot.
package registry

import (
	"fmt"
	"strings"
	"sync"

	"github.com/hrygo/divinesense/ai/agent"
)

// ToolCategory represents a category for grouping tools.
type ToolCategory string

const (
	// CategoryMemo groups memo-related tools.
	CategoryMemo ToolCategory = "memo"
	// CategorySchedule groups schedule-related tools.
	CategorySchedule ToolCategory = "schedule"
	// CategorySearch groups search/retrieval tools.
	CategorySearch ToolCategory = "search"
	// CategorySystem groups system/utility tools.
	CategorySystem ToolCategory = "system"
	// CategoryAI groups AI/LLM tools.
	CategoryAI ToolCategory = "ai"
	// CategoryCustom groups custom user-defined tools.
	CategoryCustom ToolCategory = "custom"
)

// ToolMetadata contains additional metadata about a registered tool.
type ToolMetadata struct {
	// Category is the tool's category for grouping.
	Category ToolCategory
	// Tags are optional tags for discovery (e.g., "semantic", "keyword", "recurring").
	Tags []string
	// Version is the tool version (optional).
	Version string
	// IsDeprecated indicates if the tool is deprecated. If true, ReplacedBy should be set.
	IsDeprecated bool
	// ReplacedBy is the name of the tool that replaces this one (if deprecated).
	ReplacedBy string
}

// ToolEntry wraps a tool with its metadata.
type ToolEntry struct {
	Tool     agent.ToolWithSchema
	Metadata ToolMetadata
}

// ToolRegistry manages tool registration and discovery.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolEntry
}

// Global tool registry instance.
var globalRegistry = NewToolRegistry()

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolEntry),
	}
}

// Register registers a tool globally.
// This is called during initialization to make tools available to all parrots.
func Register(name string, tool agent.ToolWithSchema) error {
	return RegisterWithMetadata(name, tool, ToolMetadata{})
}

// RegisterWithMetadata registers a tool with metadata.
func RegisterWithMetadata(name string, tool agent.ToolWithSchema, metadata ToolMetadata) error {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if _, exists := globalRegistry.tools[name]; exists {
		return fmt.Errorf("tool already registered: %s", name)
	}

	// Set default category if not specified
	if metadata.Category == "" {
		metadata.Category = inferCategory(name)
	}

	globalRegistry.tools[name] = &ToolEntry{
		Tool:     tool,
		Metadata: metadata,
	}
	return nil
}

// RegisterInCategory registers a tool in a specific category.
func RegisterInCategory(category ToolCategory, name string, tool agent.ToolWithSchema) error {
	return RegisterWithMetadata(name, tool, ToolMetadata{Category: category})
}

// Get retrieves a tool by name.
func Get(name string) (agent.ToolWithSchema, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	entry, ok := globalRegistry.tools[name]
	if !ok {
		return nil, false
	}
	return entry.Tool, true
}

// GetWithMetadata retrieves a tool entry with metadata by name.
func GetWithMetadata(name string) (*ToolEntry, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	entry, ok := globalRegistry.tools[name]
	return entry, ok
}

// List returns all registered tool names.
func List() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	names := make([]string, 0, len(globalRegistry.tools))
	for name := range globalRegistry.tools {
		names = append(names, name)
	}
	return names
}

// ListByCategory returns all tool names in a specific category.
func ListByCategory(category ToolCategory) []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	names := make([]string, 0)
	for name, entry := range globalRegistry.tools {
		if entry.Metadata.Category == category {
			names = append(names, name)
		}
	}
	return names
}

// ListWithTags returns all tool names that have any of the specified tags.
func ListWithTags(tags ...string) []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	names := make([]string, 0)
	for name, entry := range globalRegistry.tools {
		if hasAnyTag(entry.Metadata.Tags, tags) {
			names = append(names, name)
		}
	}
	return names
}

// Describe returns a formatted description of all registered tools.
func Describe() string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	if len(globalRegistry.tools) == 0 {
		return "No tools registered"
	}

	var sb strings.Builder
	sb.Grow(512)

	// Group by category
	categories := make(map[ToolCategory][]*ToolEntry)
	for _, entry := range globalRegistry.tools {
		cat := entry.Metadata.Category
		categories[cat] = append(categories[cat], entry)
	}

	// Define category order for display
	categoryOrder := []ToolCategory{
		CategoryMemo,
		CategorySchedule,
		CategorySearch,
		CategoryAI,
		CategorySystem,
		CategoryCustom,
	}

	for _, cat := range categoryOrder {
		entries := categories[cat]
		if len(entries) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("\n=== %s ===\n", strings.ToUpper(string(cat))))
		for _, entry := range entries {
			tool := entry.Tool
			sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))

			// Add tags if present
			if len(entry.Metadata.Tags) > 0 {
				sb.WriteString(fmt.Sprintf("  Tags: %s\n", strings.Join(entry.Metadata.Tags, ", ")))
			}

			// Add deprecation notice
			if entry.Metadata.IsDeprecated {
				if entry.Metadata.ReplacedBy != "" {
					sb.WriteString(fmt.Sprintf("  DEPRECATED: Use '%s' instead\n", entry.Metadata.ReplacedBy))
				} else {
					sb.WriteString("  DEPRECATED\n")
				}
			}
		}
	}

	return sb.String()
}

// DescribeTool returns a detailed description of a specific tool.
func DescribeTool(name string) string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	entry, ok := globalRegistry.tools[name]
	if !ok {
		return fmt.Sprintf("Tool not found: %s", name)
	}

	tool := entry.Tool
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Name: %s\n", tool.Name()))
	sb.WriteString(fmt.Sprintf("Description: %s\n", tool.Description()))
	sb.WriteString(fmt.Sprintf("Category: %s\n", entry.Metadata.Category))

	if len(entry.Metadata.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(entry.Metadata.Tags, ", ")))
	}

	if entry.Metadata.Version != "" {
		sb.WriteString(fmt.Sprintf("Version: %s\n", entry.Metadata.Version))
	}

	if entry.Metadata.IsDeprecated {
		sb.WriteString("Status: DEPRECATED\n")
		if entry.Metadata.ReplacedBy != "" {
			sb.WriteString(fmt.Sprintf("Replaced by: %s\n", entry.Metadata.ReplacedBy))
		}
	}

	return sb.String()
}

// Unregister removes a tool from the registry.
// This is primarily used for testing.
func Unregister(name string) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	delete(globalRegistry.tools, name)
}

// MustGet retrieves a tool by name or panics.
// This is a convenience function for initialization code.
func MustGet(name string) agent.ToolWithSchema {
	tool, ok := Get(name)
	if !ok {
		panic(fmt.Sprintf("tool not found: %s", name))
	}
	return tool
}

// Clear removes all tools from the registry.
// This is primarily used for testing.
func Clear() {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()
	globalRegistry.tools = make(map[string]*ToolEntry)
}

// Count returns the number of registered tools.
func Count() int {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()
	return len(globalRegistry.tools)
}

// CountByCategory returns the number of tools in a specific category.
func CountByCategory(category ToolCategory) int {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	count := 0
	for _, entry := range globalRegistry.tools {
		if entry.Metadata.Category == category {
			count++
		}
	}
	return count
}

// GetAllTools returns all tools as a map for easy iteration.
func GetAllTools() map[string]agent.ToolWithSchema {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make(map[string]agent.ToolWithSchema, len(globalRegistry.tools))
	for name, entry := range globalRegistry.tools {
		result[name] = entry.Tool
	}
	return result
}

// GetAllEntries returns all tool entries with metadata.
func GetAllEntries() map[string]*ToolEntry {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	result := make(map[string]*ToolEntry, len(globalRegistry.tools))
	for name, entry := range globalRegistry.tools {
		result[name] = entry
	}
	return result
}

// inferCategory infers a category from the tool name.
func inferCategory(name string) ToolCategory {
	switch {
	case strings.HasPrefix(name, "memo"):
		return CategoryMemo
	case strings.HasPrefix(name, "schedule"):
		return CategorySchedule
	case strings.Contains(name, "search") || strings.Contains(name, "query"):
		return CategorySearch
	case strings.Contains(name, "claude") || strings.Contains(name, "llm"):
		return CategoryAI
	default:
		return CategorySystem
	}
}

// hasAnyTag checks if the tool's tags contain any of the specified tags.
func hasAnyTag(toolTags []string, searchTags []string) bool {
	for _, searchTag := range searchTags {
		for _, toolTag := range toolTags {
			if strings.EqualFold(toolTag, searchTag) {
				return true
			}
		}
	}
	return false
}
