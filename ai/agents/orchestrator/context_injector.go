package orchestrator

import (
	"fmt"
	"regexp"
	"sync"
)

// Pre-compiled regex for performance (Issue #211: performance optimization)
var taskResultRegex = regexp.MustCompile(`\{\{([a-zA-Z0-9_\-]+)\.result\}\}`)

// ContextInjector handles variable substitution in task inputs.
type ContextInjector struct {
	mu sync.RWMutex
}

// NewContextInjector creates a new context injector.
func NewContextInjector() *ContextInjector {
	return &ContextInjector{}
}

// ResolveInput replaces {{task_id.result}} placeholders with actual task results.
func (ci *ContextInjector) ResolveInput(input string, tasks map[string]*Task) (string, error) {
	var err error

	// Use pre-compiled regex for performance
	resolved := taskResultRegex.ReplaceAllStringFunc(input, func(match string) string {
		// Extract task ID (group 1)
		submatches := taskResultRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match // Should not happen if regex matches
		}
		taskID := submatches[1]

		ci.mu.RLock()
		task, exists := tasks[taskID]
		ci.mu.RUnlock()

		if !exists {
			err = fmt.Errorf("reference not found: task '%s' does not exist", taskID)
			return match
		}

		if task.Status != TaskStatusCompleted {
			err = fmt.Errorf("reference invalid: task '%s' is not completed (status: %s)", taskID, task.Status)
			return match
		}

		// Simple string replacement
		// TODO: Add token limit check / summary logic here for Phase 3
		return task.Result
	})

	if err != nil {
		return "", err
	}

	return resolved, nil
}
