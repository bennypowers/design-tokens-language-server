package log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Level represents the severity of a log message
type Level int

const (
	// LevelDebug is for verbose debugging information
	LevelDebug Level = iota
	// LevelInfo is for important operational events
	LevelInfo
	// LevelWarn is for warnings that don't prevent operation
	LevelWarn
	// LevelError is for errors that may affect functionality
	LevelError
)

var (
	mu       sync.Mutex
	output   io.Writer = os.Stderr
	minLevel Level     = LevelInfo
	prefix   string    = "[DTLS]"
)

// SetOutput sets the output destination (primarily for testing)
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	output = w
}

// SetLevel sets the minimum log level to display
func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	minLevel = level
}

// GetLevel returns the current minimum log level
func GetLevel() Level {
	mu.Lock()
	defer mu.Unlock()
	return minLevel
}

// Debug logs a debug message (verbose debugging information)
func Debug(format string, args ...interface{}) {
	log(LevelDebug, format, args...)
}

// Info logs an info message (important operational events)
func Info(format string, args ...interface{}) {
	log(LevelInfo, format, args...)
}

// Warn logs a warning message (warnings that don't prevent operation)
func Warn(format string, args ...interface{}) {
	log(LevelWarn, format, args...)
}

// Error logs an error message (errors that may affect functionality)
func Error(format string, args ...interface{}) {
	log(LevelError, format, args...)
}

func log(level Level, format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if level < minLevel {
		return
	}

	// Skip logging if output is nil (e.g., during test cleanup)
	if output == nil {
		return
	}

	// Format: [DTLS] message
	fmt.Fprintf(output, prefix+" "+format+"\n", args...)
}
