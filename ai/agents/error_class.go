// Package agent provides error classification for intelligent retry logic.
// This system categorizes errors into transient (retryable), permanent (non-retryable),
// and conflict (special handling) types to improve agent reliability.
package agent

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ============================================================================
// DIP: Interface for conflict error detection
// ============================================================================

// ConflictError is an interface for detecting schedule conflict errors.
// This follows DIP (Dependency Inversion Principle) - the AI layer depends on
// an abstraction, not concrete implementations in server/store packages.
type ConflictError interface {
	error
	IsConflict() bool
}

// ============================================================================
// Error Definitions (from errors.go - merged to avoid circular dependency)
// ============================================================================

// Base error definitions for agent errors
var (
	ErrInvalidTimeFormat  = errors.New("invalid time format")
	ErrToolNotFound       = errors.New("tool not found")
	ErrParseError         = errors.New("parse error")
	ErrNetworkError       = errors.New("network error")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrScheduleConflict   = errors.New("schedule conflict")
	ErrInvalidInput       = errors.New("invalid input")
)

// ErrorClass represents the category of error for retry decisions.
type ErrorClass int

const (
	// Examples: network timeout, temporary service unavailability.
	ErrorClassTransient ErrorClass = iota

	// Examples: validation failures, permission denied, invalid input.
	ErrorClassPermanent

	// Examples: schedule overlap, duplicate booking.
	ErrorClassConflict
)

// String returns the string representation of ErrorClass.
func (e ErrorClass) String() string {
	switch e {
	case ErrorClassTransient:
		return "transient"
	case ErrorClassPermanent:
		return "permanent"
	case ErrorClassConflict:
		return "conflict"
	default:
		return "unknown"
	}
}

// ClassifiedError wraps an error with its classification and retry guidance.
type ClassifiedError struct {
	Original   error
	ActionHint string
	Class      ErrorClass
	RetryAfter time.Duration
}

// Error returns a formatted error message.
func (c *ClassifiedError) Error() string {
	if c.Original == nil {
		return fmt.Sprintf("classified error: class=%s", c.Class)
	}
	return fmt.Sprintf("%s: %v", c.Class, c.Original)
}

// Unwrap returns the original error for errors.Is/As.
func (c *ClassifiedError) Unwrap() error {
	return c.Original
}

// IsTransient returns true if the error is temporary and should be retried.
func (c *ClassifiedError) IsTransient() bool {
	return c.Class == ErrorClassTransient
}

// IsPermanent returns true if the error is non-retryable.
func (c *ClassifiedError) IsPermanent() bool {
	return c.Class == ErrorClassPermanent
}

// IsConflict returns true if the error is a conflict.
func (c *ClassifiedError) IsConflict() bool {
	return c.Class == ErrorClassConflict
}

// ClassifyError analyzes an error and determines its class and retry strategy.
func ClassifyError(err error) *ClassifiedError {
	if err == nil {
		return nil
	}

	// Check for specific known errors first

	// 1. Check for conflict errors using the ConflictError interface (DIP)
	// This works with any error type that implements IsConflict() bool
	var conflictErr ConflictError
	if errors.As(err, &conflictErr) && conflictErr.IsConflict() {
		return &ClassifiedError{
			Class:      ErrorClassConflict,
			Original:   err,
			ActionHint: "find_free_time",
		}
	}

	// 3. Check for network errors (transient)
	if isNetworkError(err) {
		return &ClassifiedError{
			Class:      ErrorClassTransient,
			Original:   err,
			RetryAfter: 2 * time.Second,
		}
	}

	// 4. Check for timeout errors (transient)
	if isTimeoutError(err) {
		return &ClassifiedError{
			Class:      ErrorClassTransient,
			Original:   err,
			RetryAfter: 3 * time.Second,
		}
	}

	// 5. Check for validation/permanent errors by error message patterns
	errMsg := strings.ToLower(err.Error())

	// Permanent: validation errors
	if strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "forbidden") ||
		strings.Contains(errMsg, "required") {
		return &ClassifiedError{
			Class:    ErrorClassPermanent,
			Original: err,
		}
	}

	// Default to permanent for unknown errors (fail safe)
	return &ClassifiedError{
		Class:    ErrorClassPermanent,
		Original: err,
	}
}

// isNetworkError checks if an error is network-related (transient).
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for common network error patterns
	errMsg := strings.ToLower(err.Error())
	networkPatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"network is unreachable",
		"no such host",
		"temporary failure",
		"dial tcp",
		"eof",
		"connection lost",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// isTimeoutError checks if an error is timeout-related (transient).
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	timeoutPatterns := []string{
		"timeout",
		"deadline exceeded",
		"context deadline exceeded",
		"i/o timeout",
		"operation timed out",
	}

	for _, pattern := range timeoutPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// ShouldRetry returns true if the error warrants a retry attempt.
func ShouldRetry(err error) bool {
	classified := ClassifyError(err)
	return classified.IsTransient()
}

// GetRetryDelay returns the suggested delay before retry, or 0 if not retryable.
func GetRetryDelay(err error) time.Duration {
	classified := ClassifyError(err)
	if classified.IsTransient() && classified.RetryAfter > 0 {
		return classified.RetryAfter
	}
	return 0
}

// GetActionHint returns the suggested action for handling the error.
func GetActionHint(err error) string {
	classified := ClassifyError(err)
	if classified.IsConflict() && classified.ActionHint != "" {
		return classified.ActionHint
	}
	return ""
}

// IsRecoverableError returns true if the error is a known type that can be recovered from.
// This is distinct from retry logic - it indicates whether the error type itself is
// fixable (e.g., invalid time format can be corrected) vs. a system-level issue.
func IsRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	// Check against our known error definitions
	return errors.Is(err, ErrInvalidTimeFormat) ||
		errors.Is(err, ErrToolNotFound) ||
		errors.Is(err, ErrParseError)
}

// IsTransientError returns true if the error is temporary and may resolve on retry.
// This maps network and service availability issues.
func IsTransientError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ErrNetworkError) ||
		errors.Is(err, ErrServiceUnavailable)
}

// MissingCapability represents an error when an expert lacks the required capability.
type MissingCapability struct {
	// Expert is the name of the expert that cannot handle the request.
	Expert string
	// MissingCapabilities lists the capabilities that the expert lacks.
	MissingCapabilities []string
	// OriginalError is the original error message from the expert.
	OriginalError error
	// Suggestion is an optional hint about which expert might help.
	Suggestion string
}

// Error returns a sanitized error message.
func (e *MissingCapability) Error() string {
	// Always return a sanitized message to prevent leaking internal error details
	return fmt.Sprintf("expert %s lacks required capabilities: %v", e.Expert, e.MissingCapabilities)
}

// Unwrap returns the original error for errors.Is/As.
func (e *MissingCapability) Unwrap() error {
	return e.OriginalError
}

// NewMissingCapability creates a new MissingCapability error.
func NewMissingCapability(expert string, missingCaps []string, originalErr error) *MissingCapability {
	return &MissingCapability{
		Expert:              expert,
		MissingCapabilities: missingCaps,
		OriginalError:       originalErr,
	}
}

// IsMissingCapability checks if an error is a MissingCapability error.
func IsMissingCapability(err error) bool {
	var missingCap *MissingCapability
	return errors.As(err, &missingCap)
}
