// Package observability provides structured logging and tracing utilities
package observability

import (
	"log"
	"os"
)

// Logger provides structured logging capabilities
type Logger struct {
	*log.Logger
}

// NewLogger creates a new structured logger with service name
func NewLogger(serviceName string) *Logger {
	return &Logger{
		Logger: log.New(
			os.Stdout,
			"["+serviceName+"] ",
			log.LstdFlags|log.Lshortfile,
		),
	}
}

// WithContext returns a child logger with additional context (for future OpenTelemetry integration)
func (l *Logger) WithContext(key, value string) *Logger {
	// TODO: Add context support when OpenTelemetry is fully integrated
	return l
}

// Info logs an info level message
func (l *Logger) Info(msg string) {
	l.Println("INFO:", msg)
}

// Warn logs a warning level message
func (l *Logger) Warn(msg string) {
	l.Println("WARN:", msg)
}

// Error logs an error level message
func (l *Logger) Error(msg string) {
	l.Println("ERROR:", msg)
}

// Debug logs a debug level message
func (l *Logger) Debug(msg string) {
	l.Println("DEBUG:", msg)
}
