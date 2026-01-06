package log

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
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
	minLevel atomic.Int32
	prefix   string = "[DTLS]"
)

func init() {
	minLevel.Store(int32(LevelInfo))
}

// SetOutput sets the output destination (primarily for testing)
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	output = w
}

// SetLevel sets the minimum log level to display
func SetLevel(level Level) {
	minLevel.Store(int32(level))
}

// GetLevel returns the current minimum log level
func GetLevel() Level {
	return Level(minLevel.Load())
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
	// Fast path: check level without lock to avoid contention for filtered messages
	if int32(level) < minLevel.Load() {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// Re-check under lock in case level changed between fast-path check and lock acquisition
	if int32(level) < minLevel.Load() {
		return
	}

	// Skip logging if output is nil (e.g., during test cleanup)
	if output == nil {
		return
	}

	// Map level to label for clarity
	levelLabel := ""
	switch level {
	case LevelDebug:
		levelLabel = "DEBUG"
	case LevelInfo:
		levelLabel = "INFO"
	case LevelWarn:
		levelLabel = "WARN"
	case LevelError:
		levelLabel = "ERROR"
	}

	// Format: [DTLS] LEVEL: message
	// Prepend prefix and level label to the args
	newArgs := make([]interface{}, 0, len(args)+2)
	newArgs = append(newArgs, prefix, levelLabel)
	newArgs = append(newArgs, args...)
	fmt.Fprintf(output, "%s %s: "+format+"\n", newArgs...)
}
