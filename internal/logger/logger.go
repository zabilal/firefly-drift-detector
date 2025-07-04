package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// LogLevel represents the severity level of a log message
type LogLevel int

const (
	// LevelDebug represents debug level logging
	LevelDebug LogLevel = iota
	// LevelInfo represents informational messages
	LevelInfo
	// LevelWarn represents warning conditions
	LevelWarn
	// LevelError represents error conditions
	LevelError
	// LevelFatal represents severe error conditions that may cause the application to exit
	LevelFatal
)

// String returns the string representation of the log level
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
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger is the main logger type
type Logger struct {
	logger *log.Logger
	level  LogLevel
}

// Config holds the configuration for the logger
type Config struct {
	Level  LogLevel
	Output io.Writer
}

var (
	// DefaultLogger is the default logger instance
	DefaultLogger *Logger
)

func init() {
	DefaultLogger = NewLogger(Config{
		Level:  LevelInfo,
		Output: os.Stdout,
	})
}

// NewLogger creates a new logger instance
func NewLogger(config Config) *Logger {
	if config.Output == nil {
		config.Output = os.Stdout
	}

	return &Logger{
		logger: log.New(config.Output, "", log.LstdFlags|log.Lmsgprefix),
		level:  config.Level,
	}
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	return l.level
}

// log is the internal logging function
func (l *Logger) log(level LogLevel, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	// Get the caller info (file and line number)
	_, file, line, ok := runtime.Caller(2) // 2 because we want the caller of the exported logging function
	caller := ""
	if ok {
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	msg := fmt.Sprintf(format, v...)
	logMsg := fmt.Sprintf("[%s] %s %s", level, caller, msg)

	l.logger.Println(logMsg)

	// For fatal errors, exit the application
	if level == LevelFatal {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(LevelDebug, format, v...)
}

// Info logs an informational message
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(LevelInfo, format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(LevelWarn, format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(LevelError, format, v...)
}

// Fatal logs a fatal error message and exits the application
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(LevelFatal, format, v...)
}

// WithFields creates a new logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	// For simplicity, we'll just add the fields to the log message prefix
	// In a more advanced implementation, you might want to use a structured logger like zap or logrus
	var fieldStrs []string
	for k, v := range fields {
		fieldStrs = append(fieldStrs, fmt.Sprintf("%s=%v", k, v))
	}
	prefix := strings.Join(fieldStrs, " ")

	return &Logger{
		logger: log.New(l.logger.Writer(), prefix+" ", l.logger.Flags()),
		level:  l.level,
	}
}

// Package-level convenience functions

// Debug logs a debug message using the default logger
func Debug(format string, v ...interface{}) {
	DefaultLogger.Debug(format, v...)
}

// Info logs an informational message using the default logger
func Info(format string, v ...interface{}) {
	DefaultLogger.Info(format, v...)
}

// Warn logs a warning message using the default logger
func Warn(format string, v ...interface{}) {
	DefaultLogger.Warn(format, v...)
}

// Error logs an error message using the default logger
func Error(format string, v ...interface{}) {
	DefaultLogger.Error(format, v...)
}

// Fatal logs a fatal error message and exits the application using the default logger
func Fatal(format string, v ...interface{}) {
	DefaultLogger.Fatal(format, v...)
}

// WithFields creates a new logger with additional fields using the default logger
func WithFields(fields map[string]interface{}) *Logger {
	return DefaultLogger.WithFields(fields)
}
