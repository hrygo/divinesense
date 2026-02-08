// Package registry extends prompt management for UniversalParrot.
package registry

import (
	"fmt"
	"sync"
)

// PromptRegistry manages prompt templates for different parrots.
// This extends the existing ai/agent/prompts.go functionality.
type PromptRegistry struct {
	mu      sync.RWMutex
	prompts map[string]*PromptTemplate
}

// PromptTemplate represents a versioned prompt template.
type PromptTemplate struct {
	Name     string
	Version  string
	Template string
	Enabled  bool
}

// Global prompt registry instance.
var promptRegistry = &PromptRegistry{
	prompts: make(map[string]*PromptTemplate),
}

// RegisterPrompt registers a prompt template.
func RegisterPrompt(name string, template *PromptTemplate) error {
	promptRegistry.mu.Lock()
	defer promptRegistry.mu.Unlock()

	if _, exists := promptRegistry.prompts[name]; exists {
		return fmt.Errorf("prompt already registered: %s", name)
	}

	promptRegistry.prompts[name] = template
	return nil
}

// GetPrompt retrieves a prompt template by name.
func GetPrompt(name string) (*PromptTemplate, bool) {
	promptRegistry.mu.RLock()
	defer promptRegistry.mu.RUnlock()

	prompt, ok := promptRegistry.prompts[name]
	return prompt, ok
}

// GetPromptTemplate returns the template string for a prompt.
func GetPromptTemplate(name string) string {
	promptRegistry.mu.RLock()
	defer promptRegistry.mu.RUnlock()

	if prompt, ok := promptRegistry.prompts[name]; ok && prompt.Enabled {
		return prompt.Template
	}
	return ""
}

// ListPrompts returns all registered prompt names.
func ListPrompts() []string {
	promptRegistry.mu.RLock()
	defer promptRegistry.mu.RUnlock()

	names := make([]string, 0, len(promptRegistry.prompts))
	for name := range promptRegistry.prompts {
		names = append(names, name)
	}
	return names
}
