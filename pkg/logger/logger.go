package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// globalLogger is the global logger instance
	globalLogger *zap.Logger
)

// Init initializes the global logger
func Init(level string, environment string) error {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if environment == "development" {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zapLevel)
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := config.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	globalLogger = logger
	return nil
}

// Get returns the global logger
func Get() *zap.Logger {
	if globalLogger == nil {
		// Fallback to a basic logger if not initialized
		config := zap.NewDevelopmentConfig()
		logger, _ := config.Build()
		return logger
	}
	return globalLogger
}

// Sync flushes any buffered log entries
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}

// WithContext returns a logger with context fields
func WithContext(ctx context.Context) *zap.Logger {
	logger := Get()
	if traceID := ctx.Value("trace_id"); traceID != nil {
		logger = logger.With(zap.String("trace_id", fmt.Sprintf("%v", traceID)))
	}
	if spanID := ctx.Value("span_id"); spanID != nil {
		logger = logger.With(zap.String("span_id", fmt.Sprintf("%v", spanID)))
	}
	return logger
}

// Debug logs a debug message
func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

// Helper functions for common field types

// String returns a zap.Field for a string
func String(key, value string) zap.Field {
	return zap.String(key, value)
}

// Int returns a zap.Field for an int
func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

// Int64 returns a zap.Field for an int64
func Int64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

// Float64 returns a zap.Field for a float64
func Float64(key string, value float64) zap.Field {
	return zap.Float64(key, value)
}

// Bool returns a zap.Field for a bool
func Bool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

// Duration returns a zap.Field for a time.Duration
func Duration(key string, value time.Duration) zap.Field {
	return zap.Duration(key, value)
}

// Time returns a zap.Field for a time.Time
func Time(key string, value time.Time) zap.Field {
	return zap.Time(key, value)
}

// ErrorField returns a zap.Field for an error
func ErrorField(err error) zap.Field {
	return zap.Error(err)
}

// Any returns a zap.Field for any value
func Any(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

// JSON returns a zap.Field that marshals the value as JSON
func JSON(key string, value interface{}) zap.Field {
	data, err := json.Marshal(value)
	if err != nil {
		return zap.String(key, fmt.Sprintf("<json-marshal-error: %v>", err))
	}
	return zap.String(key, string(data))
}

// NewTraceID generates a new trace ID (simple implementation)
func NewTraceID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

