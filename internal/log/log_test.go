package log_test

import (
	"bytes"
	"strings"
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
