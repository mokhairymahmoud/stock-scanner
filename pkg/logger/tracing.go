package logger

import (
	"context"
	"fmt"
)

// Tracing helpers for OpenTelemetry integration
// This is a placeholder for future OpenTelemetry implementation

// StartSpan creates a new span (placeholder)
func StartSpan(ctx context.Context, name string) (context.Context, func()) {
	// TODO: Implement OpenTelemetry span creation
	return ctx, func() {}
}

// WithTraceID adds a trace ID to the context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, "trace_id", traceID)
}

// WithSpanID adds a span ID to the context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, "span_id", spanID)
}

// GetTraceID retrieves the trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID := ctx.Value("trace_id"); traceID != nil {
		return fmt.Sprintf("%v", traceID)
	}
	return ""
}

// GetSpanID retrieves the span ID from context
func GetSpanID(ctx context.Context) string {
	if spanID := ctx.Value("span_id"); spanID != nil {
		return fmt.Sprintf("%v", spanID)
	}
	return ""
}

