package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// Router routes filtered alerts to the filtered stream for WebSocket Gateway
type Router struct {
	redis           storage.RedisClient
	filteredStream  string
	publishTimeout  time.Duration
}

// NewRouter creates a new alert router
func NewRouter(redis storage.RedisClient, filteredStream string, publishTimeout time.Duration) *Router {
	return &Router{
		redis:          redis,
		filteredStream: filteredStream,
		publishTimeout:  publishTimeout,
	}
}

// RouteAlert routes a filtered alert to the filtered stream
func (r *Router) RouteAlert(ctx context.Context, alert *models.Alert) error {
	// Create context with timeout
	routeCtx, cancel := context.WithTimeout(ctx, r.publishTimeout)
	defer cancel()

	// Publish to filtered stream - pass alert object directly, PublishToStream will handle JSON marshaling
	err := r.redis.PublishToStream(routeCtx, r.filteredStream, "alert", alert)
	if err != nil {
		return fmt.Errorf("failed to publish alert to filtered stream: %w", err)
	}

	logger.Debug("Routed alert to filtered stream",
		logger.String("alert_id", alert.ID),
		logger.String("rule_id", alert.RuleID),
		logger.String("symbol", alert.Symbol),
		logger.String("stream", r.filteredStream),
	)

	return nil
}

// RouteAlerts routes multiple filtered alerts
func (r *Router) RouteAlerts(ctx context.Context, alerts []*models.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	// Create context with timeout
	routeCtx, cancel := context.WithTimeout(ctx, r.publishTimeout)
	defer cancel()

	// Prepare batch messages
	// Pass alert objects directly - Redis will serialize them correctly
	messages := make([]map[string]interface{}, 0, len(alerts))
	for _, alert := range alerts {
		messages = append(messages, map[string]interface{}{
			"alert": alert, // Pass object directly, Redis will serialize
		})
	}

	if len(messages) == 0 {
		return nil
	}

	// Publish batch to filtered stream
	err := r.redis.PublishBatchToStream(routeCtx, r.filteredStream, messages)
	if err != nil {
		return fmt.Errorf("failed to publish alerts batch to filtered stream: %w", err)
	}

	logger.Debug("Routed alerts batch to filtered stream",
		logger.Int("count", len(messages)),
		logger.String("stream", r.filteredStream),
	)

	return nil
}

