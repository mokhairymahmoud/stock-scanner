package scanner

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
)

func TestAlertEmitterImpl_EmitAlert(t *testing.T) {
	redis := storage.NewMockRedisClient()
	config := DefaultAlertEmitterConfig()
	ae := NewAlertEmitter(redis, config)

	alert := &models.Alert{
		RuleID:    "rule-1",
		RuleName:  "Test Rule",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
	}

	err := ae.EmitAlert(alert)
	if err != nil {
		t.Fatalf("Failed to emit alert: %v", err)
	}

	// Verify alert ID was generated
	if alert.ID == "" {
		t.Error("Expected alert ID to be generated")
	}

	// Verify trace ID was generated
	if alert.TraceID == "" {
		t.Error("Expected trace ID to be generated")
	}

	// Verify timestamp was set
	if alert.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestAlertEmitterImpl_EmitAlert_WithID(t *testing.T) {
	redis := storage.NewMockRedisClient()
	config := DefaultAlertEmitterConfig()
	ae := NewAlertEmitter(redis, config)

	alert := &models.Alert{
		ID:        "custom-alert-id",
		RuleID:    "rule-1",
		RuleName:  "Test Rule",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
		TraceID:   "custom-trace-id",
	}

	err := ae.EmitAlert(alert)
	if err != nil {
		t.Fatalf("Failed to emit alert: %v", err)
	}

	// Verify custom IDs were preserved
	if alert.ID != "custom-alert-id" {
		t.Errorf("Expected custom alert ID, got %s", alert.ID)
	}

	if alert.TraceID != "custom-trace-id" {
		t.Errorf("Expected custom trace ID, got %s", alert.TraceID)
	}
}

func TestAlertEmitterImpl_EmitAlert_InvalidAlert(t *testing.T) {
	redis := storage.NewMockRedisClient()
	config := DefaultAlertEmitterConfig()
	ae := NewAlertEmitter(redis, config)

	// Test nil alert
	err := ae.EmitAlert(nil)
	if err == nil {
		t.Error("Expected error when emitting nil alert")
	}

	// Test invalid alert (missing symbol)
	alert := &models.Alert{
		RuleID:    "rule-1",
		RuleName:  "Test Rule",
		Symbol:    "", // Invalid
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
	}

	err = ae.EmitAlert(alert)
	if err == nil {
		t.Error("Expected error when emitting invalid alert")
	}
}

func TestAlertEmitterImpl_NewAlertEmitter(t *testing.T) {
	redis := storage.NewMockRedisClient()
	config := DefaultAlertEmitterConfig()

	// Test normal creation
	ae := NewAlertEmitter(redis, config)
	if ae == nil {
		t.Fatal("Expected alert emitter to be created")
	}

	if ae.redis != redis {
		t.Error("Expected Redis client to be set")
	}

	// Test panic with nil redis
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when redis client is nil")
		}
	}()

	NewAlertEmitter(nil, config)
}

func TestAlertEmitterImpl_GetStats(t *testing.T) {
	redis := storage.NewMockRedisClient()
	config := DefaultAlertEmitterConfig()
	ae := NewAlertEmitter(redis, config)

	stats := ae.GetStats()
	if stats.AlertsEmitted != 0 {
		t.Errorf("Expected 0 alerts emitted initially, got %d", stats.AlertsEmitted)
	}

	// Emit an alert
	alert := &models.Alert{
		RuleID:    "rule-1",
		RuleName:  "Test Rule",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
	}

	err := ae.EmitAlert(alert)
	if err != nil {
		t.Fatalf("Failed to emit alert: %v", err)
	}

	stats = ae.GetStats()
	if stats.AlertsEmitted != 1 {
		t.Errorf("Expected 1 alert emitted, got %d", stats.AlertsEmitted)
	}

	if stats.AlertsPublished != 1 {
		t.Errorf("Expected 1 alert published, got %d", stats.AlertsPublished)
	}
}

func TestAlertEmitterImpl_PubSubOnly(t *testing.T) {
	redis := storage.NewMockRedisClient()
	config := DefaultAlertEmitterConfig()
	config.StreamName = "" // Disable stream publishing
	ae := NewAlertEmitter(redis, config)

	alert := &models.Alert{
		RuleID:    "rule-1",
		RuleName:  "Test Rule",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
	}

	err := ae.EmitAlert(alert)
	if err != nil {
		t.Fatalf("Failed to emit alert: %v", err)
	}

	// Should succeed even without stream
	stats := ae.GetStats()
	if stats.AlertsEmitted != 1 {
		t.Errorf("Expected 1 alert emitted, got %d", stats.AlertsEmitted)
	}
}

func TestAlertEmitterImpl_StreamOnly(t *testing.T) {
	redis := storage.NewMockRedisClient()
	config := DefaultAlertEmitterConfig()
	config.PubSubChannel = "" // Disable pub/sub
	ae := NewAlertEmitter(redis, config)

	alert := &models.Alert{
		RuleID:    "rule-1",
		RuleName:  "Test Rule",
		Symbol:    "AAPL",
		Timestamp: time.Now(),
		Price:     150.0,
		Message:   "Test alert",
	}

	err := ae.EmitAlert(alert)
	if err != nil {
		t.Fatalf("Failed to emit alert: %v", err)
	}

	stats := ae.GetStats()
	if stats.AlertsEmitted != 1 {
		t.Errorf("Expected 1 alert emitted, got %d", stats.AlertsEmitted)
	}
}

func TestDefaultAlertEmitterConfig(t *testing.T) {
	config := DefaultAlertEmitterConfig()

	if config.PubSubChannel != "alerts" {
		t.Errorf("Expected PubSubChannel 'alerts', got '%s'", config.PubSubChannel)
	}

	if config.StreamName != "alerts" {
		t.Errorf("Expected StreamName 'alerts', got '%s'", config.StreamName)
	}

	if config.PublishTimeout != 2*time.Second {
		t.Errorf("Expected PublishTimeout 2s, got %v", config.PublishTimeout)
	}
}

func TestAlertEmitterImpl_GenerateIDs(t *testing.T) {
	redis := storage.NewMockRedisClient()
	config := DefaultAlertEmitterConfig()
	ae := NewAlertEmitter(redis, config)

	// Generate multiple IDs to ensure uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := ae.generateAlertID()
		if ids[id] {
			t.Errorf("Duplicate alert ID generated: %s", id)
		}
		ids[id] = true
	}

	// Generate trace IDs
	traceIDs := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := ae.generateTraceID()
		if traceIDs[id] {
			t.Errorf("Duplicate trace ID generated: %s", id)
		}
		traceIDs[id] = true
	}
}

func TestAlertEmitterImpl_Concurrency(t *testing.T) {
	redis := storage.NewMockRedisClient()
	config := DefaultAlertEmitterConfig()
	ae := NewAlertEmitter(redis, config)

	// Test concurrent alert emission
	done := make(chan bool)
	alertCount := 100

	for i := 0; i < alertCount; i++ {
		go func(idx int) {
			alert := &models.Alert{
				RuleID:    "rule-1",
				RuleName:  "Test Rule",
				Symbol:    "AAPL",
				Timestamp: time.Now(),
				Price:     150.0 + float64(idx),
				Message:   "Test alert",
			}

			err := ae.EmitAlert(alert)
			if err != nil {
				t.Errorf("Failed to emit alert: %v", err)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < alertCount; i++ {
		<-done
	}

	// Verify stats
	stats := ae.GetStats()
	if stats.AlertsEmitted != int64(alertCount) {
		t.Errorf("Expected %d alerts emitted, got %d", alertCount, stats.AlertsEmitted)
	}
}

