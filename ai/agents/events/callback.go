// Package events provides event callback types for the agent system.
// This package centralizes callback definitions to follow DRY principle.
package events

import (
	"log/slog"
	"runtime/debug"
)

// Callback is the unified event callback type.
// It receives an event type string and arbitrary event data.
type Callback func(eventType string, eventData any) error

// SafeCallback is a callback variant that does not propagate errors.
// Errors are logged internally instead of being returned to callers.
type SafeCallback func(eventType string, eventData any)

// NoopCallback is a callback that does nothing.
var NoopCallback Callback = func(string, any) error { return nil }

// WrapSafe converts a Callback to a SafeCallback.
// Errors from the original callback are logged but not propagated.
// Returns nil if the input callback is nil.
func WrapSafe(cb Callback) SafeCallback {
	if cb == nil {
		return nil
	}
	return func(eventType string, eventData any) {
		if err := cb(eventType, eventData); err != nil {
			slog.Warn("event callback error (swallowed)",
				"event_type", eventType,
				"error", err,
				"stack", string(debug.Stack()))
		}
	}
}
