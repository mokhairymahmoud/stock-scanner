package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// AlertEmitterConfig holds configuration for the alert emitter
type AlertEmitterConfig struct {
	PubSubChannel  string        // Redis pub/sub channel (default: "alerts")
	StreamName     string        // Redis stream name (optional, for persistence)
	PublishTimeout time.Duration // Timeout for publishing alerts
	TraceIDHeader  string        // Header name for trace ID (default: "X-Trace-ID")
}

// DefaultAlertEmitterConfig returns default configuration
func DefaultAlertEmitterConfig() AlertEmitterConfig {
	return AlertEmitterConfig{
		PubSubChannel:  "alerts",
		StreamName:     "alerts", // Optional: can be empty to disable stream publishing
		PublishTimeout: 2 * time.Second,
		TraceIDHeader:  "X-Trace-ID",
	}
}

// AlertEmitterImpl implements the AlertEmitter interface
// Publishes alerts to Redis pub/sub and optionally to Redis streams
type AlertEmitterImpl struct {
	config  AlertEmitterConfig
	redis   storage.RedisClient
	mu      sync.RWMutex
	running bool
	stats   AlertEmitterStats
}

// AlertEmitterStats holds statistics about alert emission
type AlertEmitterStats struct {
	AlertsEmitted   int64
	AlertsPublished int64
	AlertsFailed    int64
	LastAlertTime   time.Time
	mu              sync.RWMutex
}

// NewAlertEmitter creates a new alert emitter
func NewAlertEmitter(redis storage.RedisClient, config AlertEmitterConfig) *AlertEmitterImpl {
	if redis == nil {
		panic("redis client cannot be nil")
	}

	return &AlertEmitterImpl{
		config: config,
		redis:  redis,
		stats:  AlertEmitterStats{},
	}
}

// EmitAlert emits an alert to Redis
func (ae *AlertEmitterImpl) EmitAlert(alert *models.Alert) error {
	if alert == nil {
		return fmt.Errorf("alert cannot be nil")
	}

	// Generate alert ID if not set (before validation)
	if alert.ID == "" {
		alert.ID = ae.generateAlertID()
	}

	// Set timestamp if not set
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	// Generate trace ID if not set
	if alert.TraceID == "" {
		alert.TraceID = ae.generateTraceID()
	}

	// Validate alert (after setting required fields)
	if err := alert.Validate(); err != nil {
		return fmt.Errorf("invalid alert: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ae.config.PublishTimeout)
	defer cancel()

	// Marshal alert to JSON for pub/sub
	alertJSON, err := json.Marshal(alert)
	if err != nil {
		ae.incrementFailed()
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	// Publish to pub/sub channel (real-time delivery)
	if ae.config.PubSubChannel != "" {
		err = ae.redis.Publish(ctx, ae.config.PubSubChannel, string(alertJSON))
		if err != nil {
			logger.Error("Failed to publish alert to pub/sub",
				logger.ErrorField(err),
				logger.String("channel", ae.config.PubSubChannel),
				logger.String("alert_id", alert.ID),
			)
			// Don't fail the whole operation if pub/sub fails
		} else {
			logger.Debug("Published alert to pub/sub",
				logger.String("channel", ae.config.PubSubChannel),
				logger.String("alert_id", alert.ID),
				logger.String("rule_id", alert.RuleID),
				logger.String("symbol", alert.Symbol),
			)
		}
	}

	// Publish to Redis stream (optional, for persistence)
	// Pass the alert object directly - PublishToStream will handle JSON marshaling
	if ae.config.StreamName != "" {
		err = ae.redis.PublishToStream(ctx, ae.config.StreamName, "alert", alert)
		if err != nil {
			logger.Error("Failed to publish alert to stream",
				logger.ErrorField(err),
				logger.String("stream", ae.config.StreamName),
				logger.String("alert_id", alert.ID),
			)
			ae.incrementFailed()
			return fmt.Errorf("failed to publish alert to stream: %w", err)
		}

		logger.Debug("Published alert to stream",
			logger.String("stream", ae.config.StreamName),
			logger.String("alert_id", alert.ID),
		)
	}

	ae.incrementEmitted()
	return nil
}

// GetStats returns current alert emitter statistics
func (ae *AlertEmitterImpl) GetStats() AlertEmitterStats {
	ae.stats.mu.RLock()
	defer ae.stats.mu.RUnlock()

	// Return a copy
	return AlertEmitterStats{
		AlertsEmitted:   ae.stats.AlertsEmitted,
		AlertsPublished: ae.stats.AlertsPublished,
		AlertsFailed:    ae.stats.AlertsFailed,
		LastAlertTime:   ae.stats.LastAlertTime,
	}
}

// generateAlertID generates a unique alert ID
func (ae *AlertEmitterImpl) generateAlertID() string {
	return uuid.New().String()
}

// generateTraceID generates a unique trace ID
func (ae *AlertEmitterImpl) generateTraceID() string {
	return uuid.New().String()
}

// incrementEmitted increments the emitted alert counter
func (ae *AlertEmitterImpl) incrementEmitted() {
	ae.stats.mu.Lock()
	defer ae.stats.mu.Unlock()
	ae.stats.AlertsEmitted++
	ae.stats.AlertsPublished++
	ae.stats.LastAlertTime = time.Now()
}

// incrementFailed increments the failed alert counter
func (ae *AlertEmitterImpl) incrementFailed() {
	ae.stats.mu.Lock()
	defer ae.stats.mu.Unlock()
	ae.stats.AlertsFailed++
}
