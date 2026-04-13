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
