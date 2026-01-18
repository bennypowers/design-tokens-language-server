package log_test

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"bennypowers.dev/dtls/internal/log"
	"github.com/stretchr/testify/assert"
)

func TestLogLevels(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil) // Reset after test

	t.Run("Info level logs Info, Warn, Error but not Debug", func(t *testing.T) {
		buf.Reset()
		log.SetLevel(log.LevelInfo)

		log.Debug("debug message")
		log.Info("info message")
		log.Warn("warn message")
		log.Error("error message")

		output := buf.String()
		assert.NotContains(t, output, "debug message", "Debug should not be logged at Info level")
		assert.Contains(t, output, "info message", "Info should be logged")
		assert.Contains(t, output, "warn message", "Warn should be logged")
		assert.Contains(t, output, "error message", "Error should be logged")
	})

	t.Run("Error level only logs Error", func(t *testing.T) {
		buf.Reset()
		log.SetLevel(log.LevelError)

		log.Debug("debug message")
		log.Info("info message")
		log.Warn("warn message")
		log.Error("error message")

		output := buf.String()
		assert.NotContains(t, output, "debug message")
		assert.NotContains(t, output, "info message")
		assert.NotContains(t, output, "warn message")
		assert.Contains(t, output, "error message", "Error should be logged")
	})

	t.Run("Debug level logs everything", func(t *testing.T) {
		buf.Reset()
		log.SetLevel(log.LevelDebug)

		log.Debug("debug message")
		log.Info("info message")
		log.Warn("warn message")
		log.Error("error message")

		output := buf.String()
		assert.Contains(t, output, "debug message", "Debug should be logged")
		assert.Contains(t, output, "info message", "Info should be logged")
		assert.Contains(t, output, "warn message", "Warn should be logged")
		assert.Contains(t, output, "error message", "Error should be logged")
	})
}

func TestLogFormat(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetLevel(log.LevelInfo)
	defer log.SetOutput(nil)

	t.Run("Messages include [DTLS] prefix", func(t *testing.T) {
		buf.Reset()
		log.Info("test message")

		output := buf.String()
		assert.Contains(t, output, "[DTLS]", "Should have [DTLS] prefix")
		assert.Contains(t, output, "test message")
	})

	t.Run("Format strings work correctly", func(t *testing.T) {
		buf.Reset()
		log.Info("Publishing diagnostics for: %s", "file:///test.json")

		output := buf.String()
		assert.Contains(t, output, "Publishing diagnostics for: file:///test.json")
	})

	t.Run("Each log message ends with newline", func(t *testing.T) {
		buf.Reset()
		log.Info("message 1")
		log.Info("message 2")

		lines := strings.Split(buf.String(), "\n")
		// Should have 2 messages plus empty string after final newline
		assert.GreaterOrEqual(t, len(lines), 2)
		assert.Contains(t, lines[0], "message 1")
		assert.Contains(t, lines[1], "message 2")
	})

	t.Run("Messages include level labels", func(t *testing.T) {
		buf.Reset()
		log.SetLevel(log.LevelDebug)

		log.Debug("debug")
		log.Info("info")
		log.Warn("warn")
		log.Error("error")

		output := buf.String()
		assert.Contains(t, output, "DEBUG:", "Should include DEBUG level")
		assert.Contains(t, output, "INFO:", "Should include INFO level")
		assert.Contains(t, output, "WARN:", "Should include WARN level")
		assert.Contains(t, output, "ERROR:", "Should include ERROR level")
	})
}

func TestGetLevel(t *testing.T) {
	// Save original level
	originalLevel := log.GetLevel()
	defer log.SetLevel(originalLevel)

	log.SetLevel(log.LevelDebug)
	assert.Equal(t, log.LevelDebug, log.GetLevel())

	log.SetLevel(log.LevelError)
	assert.Equal(t, log.LevelError, log.GetLevel())
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    log.Level
		expected string
	}{
		{log.LevelDebug, "LevelDebug"},
		{log.LevelInfo, "LevelInfo"},
		{log.LevelWarn, "LevelWarn"},
		{log.LevelError, "LevelError"},
		{log.Level(99), "Level(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)
	log.SetLevel(log.LevelDebug)

	// Test concurrent logging from multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				log.Info("message from goroutine %d iteration %d", id, j)
				log.Debug("debug from goroutine %d iteration %d", id, j)
			}
		}(i)
	}

	wg.Wait()

	// Verify all messages are present without corruption
	output := buf.String()
	lines := strings.Split(output, "\n")

	// Should have numGoroutines * messagesPerGoroutine * 2 (Info + Debug) messages
	// Plus empty lines from final newlines
	expectedMessages := numGoroutines * messagesPerGoroutine * 2
	nonEmptyLines := 0
	for _, line := range lines {
		if line != "" {
			nonEmptyLines++
		}
	}

	assert.Equal(t, expectedMessages, nonEmptyLines, "All messages should be logged without loss")

	// Verify no message corruption (each line should have [DTLS] prefix and level)
	for _, line := range lines {
		if line != "" {
			assert.Contains(t, line, "[DTLS]", "Each line should have prefix")
			// Should have either INFO: or DEBUG:
			hasLevel := strings.Contains(line, "INFO:") || strings.Contains(line, "DEBUG:")
			assert.True(t, hasLevel, "Each line should have level label")
		}
	}
}

func TestConcurrentLevelChanges(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	// Test changing log level while logging is happening
	var wg sync.WaitGroup

	// Goroutine that constantly changes log level
	wg.Add(1)
	go func() {
		defer wg.Done()
		levels := []log.Level{log.LevelDebug, log.LevelInfo, log.LevelWarn, log.LevelError}
		for i := 0; i < 100; i++ {
			log.SetLevel(levels[i%len(levels)])
		}
	}()

	// Goroutines that log messages
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				log.Debug("debug %d-%d", id, j)
				log.Info("info %d-%d", id, j)
				log.Warn("warn %d-%d", id, j)
				log.Error("error %d-%d", id, j)
			}
		}(i)
	}

	wg.Wait()

	// Verify no corruption (all lines should be well-formed)
	output := buf.String()
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line != "" {
			assert.Contains(t, line, "[DTLS]", "Each line should have prefix")
		}
	}

	// Verify we can still get/set level after concurrent operations
	log.SetLevel(log.LevelInfo)
	assert.Equal(t, log.LevelInfo, log.GetLevel())
}
