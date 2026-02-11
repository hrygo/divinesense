// Package logging provides structured logging utilities for AI modules.
package logging

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity level of a log entry.
type LogLevel int

const (
	// LevelDebug is for detailed debugging information.
	LevelDebug LogLevel = iota
	// LevelInfo is for general informational messages.
	LevelInfo
	// LevelWarn is for warning messages.
	LevelWarn
	// LevelError is for error messages.
	LevelError
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging with context support.
type Logger struct {
	mu      sync.RWMutex
	handler slog.Handler
	level   LogLevel
	fields  map[string]interface{}
}

// Default logger instance.
var defaultLogger *Logger

func init() {
	// Initialize default logger with JSON handler for production
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	defaultLogger = NewLogger(handler)
}

// NewLogger creates a new logger with the given handler.
func NewLogger(h slog.Handler) *Logger {
	if h == nil {
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	return &Logger{
		handler: h,
		level:   LevelInfo,
		fields:  make(map[string]interface{}),
	}
}

// WithLevel returns a new logger with the specified minimum level.
func (l *Logger) WithLevel(level LogLevel) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		handler: l.handler,
		level:   level,
		fields:  make(map[string]interface{}),
	}
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// WithField returns a new logger with an additional field.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		handler: l.handler,
		level:   l.level,
		fields:  make(map[string]interface{}),
	}
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	// Add new field
	newLogger.fields[key] = value
	return newLogger
}

// WithFields returns a new logger with additional fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		handler: l.handler,
		level:   l.level,
		fields:  make(map[string]interface{}),
	}
	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.log(LevelDebug, msg, args...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...any) {
	l.log(LevelInfo, msg, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.log(LevelWarn, msg, args...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.log(LevelError, msg, args...)
}

// log handles the actual logging logic.
func (l *Logger) log(level LogLevel, msg string, args ...any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Build slog attributes from fields
	attrs := make([]slog.Attr, 0, len(l.fields)+len(args)/2)
	for k, v := range l.fields {
		attrs = append(attrs, slog.Any(k, v))
	}

	// Handle variadic args as key-value pairs
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key, _ := args[i].(string)
			attrs = append(attrs, slog.Any(key, args[i+1]))
		}
	}

	// Create the log record
	record := slog.NewRecord(time.Now(), slog.Level(level), msg, 0)
	record.AddAttrs(attrs...)

	// Handle using the underlying handler
	_ = l.handler.Handle(context.Background(), record)
}

// FromContext extracts the logger from context.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return l
	}
	return defaultLogger
}

// ToContext adds the logger to context.
func ToContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, l)
}

type loggerKey struct{}

// Package-level convenience functions using default logger.

// Debug logs a debug message using the default logger.
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs an info message using the default logger.
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs a warning message using the default logger.
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs an error message using the default logger.
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// SetLevel sets the minimum log level for the default logger.
func SetLevel(level LogLevel) {
	defaultLogger = defaultLogger.WithLevel(level)
}
