// Package agent provides recoverable error definitions for AI agents.
// These errors are designed to work with the error recovery system.
package agent

import "errors"

// Recoverable error types for the error recovery system.
// These errors can be automatically handled by ErrorRecovery.
var (
	// ErrInvalidTimeFormat indicates the time expression could not be parsed.
	// Recovery: Attempt to normalize the time expression and retry.
	ErrInvalidTimeFormat = errors.New("invalid time format")

	// ErrToolNotFound indicates the requested tool does not exist.
	// Recovery: Re-route to find an appropriate tool.
	ErrToolNotFound = errors.New("tool not found")

	// ErrParseError indicates the input could not be parsed.
	// Recovery: Simplify the input and retry.
	ErrParseError = errors.New("parse error")

	// ErrNetworkError indicates a network-related failure.
	// Recovery: Not automatically recoverable, return friendly message.
	ErrNetworkError = errors.New("network error")

	// ErrServiceUnavailable indicates the service is temporarily unavailable.
	// Recovery: Not automatically recoverable, return friendly message.
	ErrServiceUnavailable = errors.New("service unavailable")

	// ErrInvalidInput indicates the user input is invalid.
	// Recovery: Not automatically recoverable, return friendly message.
	ErrInvalidInput = errors.New("invalid input")

	// ErrScheduleConflict indicates a schedule time conflict.
	// Recovery: Not automatically recoverable, suggest alternatives.
	ErrScheduleConflict = errors.New("schedule conflict")
)

// IsRecoverableError checks if the error can be automatically recovered.
func IsRecoverableError(err error) bool {
	return errors.Is(err, ErrInvalidTimeFormat) ||
		errors.Is(err, ErrToolNotFound) ||
		errors.Is(err, ErrParseError)
}

// IsTransientError checks if the error is transient and might succeed on retry.
func IsTransientError(err error) bool {
	return errors.Is(err, ErrNetworkError) ||
		errors.Is(err, ErrServiceUnavailable)
}
