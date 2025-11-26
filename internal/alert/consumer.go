package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// Consumer consumes alerts from Redis stream and processes them
type Consumer struct {
	config        config.AlertConfig
	redis         storage.RedisClient
	deduplicator  *Deduplicator
	filter        *UserFilter
	persister     *AlertPersister
	router        *Router
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	running       bool
	stats         ConsumerStats
}

// ConsumerStats holds statistics about the consumer
type ConsumerStats struct {
	AlertsReceived    int64
	AlertsProcessed   int64
	AlertsDeduplicated int64
	AlertsFiltered    int64
	AlertsRouted      int64
	AlertsFailed      int64
	LastAlertTime     time.Time
	mu                sync.RWMutex
}

// NewConsumer creates a new alert consumer
func NewConsumer(
	config config.AlertConfig,
	redis storage.RedisClient,
	deduplicator *Deduplicator,
	filter *UserFilter,
	persister *AlertPersister,
	router *Router,
) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		config:       config,
		redis:        redis,
		deduplicator: deduplicator,
		filter:       filter,
		persister:    persister,
		router:       router,
		ctx:          ctx,
		cancel:       cancel,
		stats:        ConsumerStats{},
	}
}

// Start starts consuming alerts from the stream
func (c *Consumer) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.running = true
	c.mu.Unlock()

	logger.Info("Starting alert consumer",
		logger.String("stream", c.config.StreamName),
		logger.String("group", c.config.ConsumerGroup),
		logger.String("consumer", "alert-service-1"),
	)

	// Start consuming from stream
	messageChan, err := c.redis.ConsumeFromStream(
		c.ctx,
		c.config.StreamName,
		c.config.ConsumerGroup,
		"alert-service-1",
	)
	if err != nil {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		return fmt.Errorf("failed to start consuming from stream: %w", err)
	}

	c.wg.Add(1)
	go c.processMessages(messageChan)

	return nil
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	logger.Info("Stopping alert consumer")
	c.cancel()
	c.wg.Wait()
	logger.Info("Alert consumer stopped")
}

// processMessages processes messages from the stream
func (c *Consumer) processMessages(messageChan <-chan storage.StreamMessage) {
	defer c.wg.Done()

	batch := make([]storage.StreamMessage, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.ProcessTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			// Process remaining batch before exiting
			if len(batch) > 0 {
				c.processBatch(batch)
			}
			return

		case msg, ok := <-messageChan:
			if !ok {
				logger.Warn("Message channel closed")
				// Process remaining batch
				if len(batch) > 0 {
					c.processBatch(batch)
				}
				return
			}

			batch = append(batch, msg)
			c.incrementReceived()

			// Process batch if it's full
			if len(batch) >= c.config.BatchSize {
				c.processBatch(batch)
				batch = batch[:0] // Clear batch
			}

		case <-ticker.C:
			// Process batch on timeout
			if len(batch) > 0 {
				c.processBatch(batch)
				batch = batch[:0] // Clear batch
			}
		}
	}
}

// processBatch processes a batch of messages
func (c *Consumer) processBatch(messages []storage.StreamMessage) {
	if len(messages) == 0 {
		return
	}

	processed := make([]string, 0, len(messages)) // Message IDs to acknowledge
	failed := make([]string, 0)                  // Message IDs that failed

	for _, msg := range messages {
		// Deserialize alert
		alert, err := c.deserializeAlert(msg)
		if err != nil {
			logger.Error("Failed to deserialize alert",
				logger.ErrorField(err),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			c.incrementFailed()
			continue
		}

		// Process alert
		shouldAck, err := c.processAlert(alert)
		if err != nil {
			logger.Error("Failed to process alert",
				logger.ErrorField(err),
				logger.String("alert_id", alert.ID),
				logger.String("message_id", msg.ID),
			)
			failed = append(failed, msg.ID)
			c.incrementFailed()
			continue
		}

		if shouldAck {
			processed = append(processed, msg.ID)
			c.incrementProcessed()
		} else {
			// Alert was filtered/duplicated, still acknowledge
			processed = append(processed, msg.ID)
		}
	}

	// Acknowledge successfully processed messages
	if len(processed) > 0 {
		c.acknowledgeMessages(processed)
	}

	// Log failed messages (they will be retried by consumer group)
	if len(failed) > 0 {
		logger.Warn("Some alerts failed to process",
			logger.Int("failed_count", len(failed)),
		)
	}
}

// processAlert processes a single alert through the pipeline
// Returns true if alert should be acknowledged, false if it was filtered/duplicated
func (c *Consumer) processAlert(alert *models.Alert) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.ProcessTimeout)
	defer cancel()

	// Step 1: Deduplication
	isDuplicate, err := c.deduplicator.IsDuplicate(ctx, alert)
	if err != nil {
		return false, fmt.Errorf("deduplication failed: %w", err)
	}
	if isDuplicate {
		c.incrementDeduplicated()
		return true, nil // Acknowledge but don't process further
	}

	// Step 2: User filtering
	passFilter, err := c.filter.FilterAlert(ctx, alert)
	if err != nil {
		return false, fmt.Errorf("filtering failed: %w", err)
	}
	if !passFilter {
		c.incrementFiltered()
		return true, nil // Acknowledge but don't process further
	}

	// Step 3: Persist alert (async, non-blocking)
	err = c.persister.WriteAlerts(ctx, []*models.Alert{alert})
	if err != nil {
		logger.Warn("Failed to persist alert",
			logger.ErrorField(err),
			logger.String("alert_id", alert.ID),
		)
		// Don't fail the operation, continue to routing
	}

	// Step 4: Route to filtered stream
	err = c.router.RouteAlert(ctx, alert)
	if err != nil {
		return false, fmt.Errorf("routing failed: %w", err)
	}

	c.incrementRouted()
	return true, nil
}

// deserializeAlert deserializes a stream message into an Alert
func (c *Consumer) deserializeAlert(msg storage.StreamMessage) (*models.Alert, error) {
	// Try to get alert from message values
	alertValue, ok := msg.Values["alert"]
	if !ok {
		return nil, fmt.Errorf("alert field not found in message")
	}

	alertStr, ok := alertValue.(string)
	if !ok {
		return nil, fmt.Errorf("alert field is not a string")
	}

	var alert models.Alert
	if err := json.Unmarshal([]byte(alertStr), &alert); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alert: %w", err)
	}

	return &alert, nil
}

// acknowledgeMessages acknowledges processed messages
func (c *Consumer) acknowledgeMessages(messageIDs []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, id := range messageIDs {
		err := c.redis.AcknowledgeMessage(ctx, c.config.StreamName, c.config.ConsumerGroup, id)
		if err != nil {
			logger.Warn("Failed to acknowledge message",
				logger.ErrorField(err),
				logger.String("message_id", id),
			)
		}
	}
}

// GetStats returns current consumer statistics
func (c *Consumer) GetStats() ConsumerStats {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	// Return a copy
	return ConsumerStats{
		AlertsReceived:    c.stats.AlertsReceived,
		AlertsProcessed:   c.stats.AlertsProcessed,
		AlertsDeduplicated: c.stats.AlertsDeduplicated,
		AlertsFiltered:    c.stats.AlertsFiltered,
		AlertsRouted:      c.stats.AlertsRouted,
		AlertsFailed:      c.stats.AlertsFailed,
		LastAlertTime:     c.stats.LastAlertTime,
	}
}

// Stats increment methods
func (c *Consumer) incrementReceived() {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	c.stats.AlertsReceived++
	c.stats.LastAlertTime = time.Now()
}

func (c *Consumer) incrementProcessed() {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	c.stats.AlertsProcessed++
}

func (c *Consumer) incrementDeduplicated() {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	c.stats.AlertsDeduplicated++
}

func (c *Consumer) incrementFiltered() {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	c.stats.AlertsFiltered++
}


func (c *Consumer) incrementRouted() {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	c.stats.AlertsRouted++
}

func (c *Consumer) incrementFailed() {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()
	c.stats.AlertsFailed++
}

