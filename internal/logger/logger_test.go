package logger_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yourusername/driftdetector/internal/logger"
)

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name     string
		level    logger.LogLevel
		logFunc  func(*logger.Logger, string, ...interface{})
		expected bool
	}{
		{
			name:     "debug level with debug log",
			level:    logger.LevelDebug,
			logFunc:  (*logger.Logger).Debug,
			expected: true,
		},
		{
			name:     "info level with debug log",
			level:    logger.LevelInfo,
			logFunc:  (*logger.Logger).Debug,
			expected: false,
		},
		{
			name:     "info level with info log",
			level:    logger.LevelInfo,
			logFunc:  (*logger.Logger).Info,
			expected: true,
		},
		{
			name:     "warn level with info log",
			level:    logger.LevelWarn,
			logFunc:  (*logger.Logger).Info,
			expected: false,
		},
		{
			name:     "error level with warn log",
			level:    logger.LevelError,
			logFunc:  (*logger.Logger).Warn,
			expected: false,
		},
		{
			name:     "fatal level with error log",
			level:    logger.LevelFatal,
			logFunc:  (*logger.Logger).Error,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := logger.NewLogger(logger.Config{
				Level:  tt.level,
				Output: &buf,
			})

			tt.logFunc(logger, "test message")

			output := buf.String()
			if tt.expected {
				assert.Contains(t, output, "test message")
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestLogger_OutputFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := logger.NewLogger(logger.Config{
		Level:  logger.LevelInfo,
		Output: &buf,
	})

	logger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "[INFO]")
	assert.Contains(t, output, "test message")
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := logger.NewLogger(logger.Config{
		Level:  logger.LevelInfo,
		Output: &buf,
	})

	fields := map[string]interface{}{
		"request_id": "12345",
		"user":      "testuser",
	}

	logger.WithFields(fields).Info("user logged in")

	output := buf.String()
	assert.Contains(t, output, "user=testuser")
	assert.Contains(t, output, "request_id=12345")
	assert.Contains(t, output, "user logged in")
}

func TestDefaultLogger(t *testing.T) {
	// This test just ensures the default logger doesn't panic
	// We can't easily test the output since it goes to stdout
	logger.Info("test default logger")
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    logger.LogLevel
		expected string
	}{
		{logger.LevelDebug, "DEBUG"},
		{logger.LevelInfo, "INFO"},
		{logger.LevelWarn, "WARN"},
		{logger.LevelError, "ERROR"},
		{logger.LevelFatal, "FATAL"},
		{logger.LogLevel(999), "UNKNOWN"}, // Test unknown level
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestLogger_Concurrent(t *testing.T) {
	var buf bytes.Buffer
	logger := logger.NewLogger(logger.Config{
		Level:  logger.LevelDebug,
		Output: &buf,
	})

	// Run multiple goroutines that log concurrently
	done := make(chan bool)
	count := 100

	for i := 0; i < count; i++ {
		go func(n int) {
			logger.Info("message %d", n)
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < count; i++ {
		<-done
	}

	// Verify all messages were logged
	output := buf.String()
	for i := 0; i < count; i++ {
		expected := fmt.Sprintf("message %d", i)
		assert.Contains(t, output, expected)
	}
}
