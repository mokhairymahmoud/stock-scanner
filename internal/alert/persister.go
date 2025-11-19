package alert

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// AlertPersister handles persisting alerts to TimescaleDB
type AlertPersister struct {
	db          *sql.DB
	dbConfig    config.DatabaseConfig
	writeConfig WriteConfig

	// Write queue
	writeQueue chan []*models.Alert
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
}

// WriteConfig holds configuration for write operations
type WriteConfig struct {
	BatchSize   int
	Interval    time.Duration
	QueueSize   int
	MaxRetries  int
	RetryDelay  time.Duration
}

// NewAlertPersister creates a new alert persister
func NewAlertPersister(dbConfig config.DatabaseConfig, writeConfig WriteConfig) (*AlertPersister, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Database,
		dbConfig.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(dbConfig.MaxConnections)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	clientCtx, clientCancel := context.WithCancel(context.Background())

	persister := &AlertPersister{
		db:          db,
		dbConfig:    dbConfig,
		writeConfig: writeConfig,
		writeQueue:  make(chan []*models.Alert, writeConfig.QueueSize),
		ctx:         clientCtx,
		cancel:      clientCancel,
	}

	logger.Info("Alert persister initialized",
		logger.String("host", dbConfig.Host),
		logger.Int("port", dbConfig.Port),
		logger.String("database", dbConfig.Database),
	)

	return persister, nil
}

// Start starts the write queue processor
func (p *AlertPersister) Start() error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("persister is already running")
	}
	p.running = true
	p.mu.Unlock()

	p.wg.Add(1)
	go p.processWriteQueue()

	logger.Info("Alert persister started")
	return nil
}

// Stop stops the write queue processor
func (p *AlertPersister) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	logger.Info("Stopping alert persister")
	p.cancel()
	close(p.writeQueue)
	p.wg.Wait()
	logger.Info("Alert persister stopped")
}

// WriteAlerts enqueues alerts for async writing
func (p *AlertPersister) WriteAlerts(ctx context.Context, alerts []*models.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	// Validate alerts
	validAlerts := make([]*models.Alert, 0, len(alerts))
	for _, alert := range alerts {
		if err := alert.Validate(); err != nil {
			logger.Warn("Invalid alert, skipping",
				logger.ErrorField(err),
				logger.String("alert_id", alert.ID),
			)
			continue
		}
		validAlerts = append(validAlerts, alert)
	}

	if len(validAlerts) == 0 {
		return nil
	}

	// Try to enqueue (non-blocking with timeout)
	select {
	case p.writeQueue <- validAlerts:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
		// Queue might be full, log warning but still try
		logger.Warn("Write queue may be full, attempting to enqueue",
			logger.Int("queue_depth", len(p.writeQueue)),
			logger.Int("alerts_count", len(validAlerts)),
		)
		select {
		case p.writeQueue <- validAlerts:
			return nil
		default:
			return fmt.Errorf("write queue is full")
		}
	}
}

// processWriteQueue processes the write queue
func (p *AlertPersister) processWriteQueue() {
	defer p.wg.Done()

	batch := make([]*models.Alert, 0, p.writeConfig.BatchSize)
	ticker := time.NewTicker(p.writeConfig.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			// Process remaining batch before exiting
			if len(batch) > 0 {
				p.writeBatch(batch)
			}
			return

		case alerts, ok := <-p.writeQueue:
			if !ok {
				// Channel closed, process remaining batch
				if len(batch) > 0 {
					p.writeBatch(batch)
				}
				return
			}

			batch = append(batch, alerts...)

			// Write batch if it's full
			if len(batch) >= p.writeConfig.BatchSize {
				p.writeBatch(batch)
				batch = batch[:0] // Clear batch
			}

		case <-ticker.C:
			// Write batch on interval
			if len(batch) > 0 {
				p.writeBatch(batch)
				batch = batch[:0] // Clear batch
			}
		}
	}
}

// writeBatch writes a batch of alerts to the database
func (p *AlertPersister) writeBatch(alerts []*models.Alert) {
	if len(alerts) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Retry logic
	var err error
	for attempt := 0; attempt < p.writeConfig.MaxRetries; attempt++ {
		err = p.insertBatch(ctx, alerts)
		if err == nil {
			logger.Debug("Successfully wrote alerts batch",
				logger.Int("count", len(alerts)),
			)
			return
		}

		if attempt < p.writeConfig.MaxRetries-1 {
			logger.Warn("Failed to write alerts batch, retrying",
				logger.ErrorField(err),
				logger.Int("attempt", attempt+1),
				logger.Int("max_retries", p.writeConfig.MaxRetries),
			)
			time.Sleep(p.writeConfig.RetryDelay)
		}
	}

	logger.Error("Failed to write alerts batch after retries",
		logger.ErrorField(err),
		logger.Int("count", len(alerts)),
	)
}

// insertBatch inserts a batch of alerts into the database
func (p *AlertPersister) insertBatch(ctx context.Context, alerts []*models.Alert) error {
	query := `
		INSERT INTO alert_history (id, rule_id, rule_name, symbol, timestamp, price, message, metadata, trace_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
	`

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

		for _, alert := range alerts {
		// Serialize metadata to JSON
		var metadataJSON []byte
		if alert.Metadata != nil && len(alert.Metadata) > 0 {
			metadataJSON, err = json.Marshal(alert.Metadata)
			if err != nil {
				logger.Warn("Failed to marshal metadata, using empty object",
					logger.ErrorField(err),
					logger.String("alert_id", alert.ID),
				)
				metadataJSON = []byte("{}")
			}
		} else {
			metadataJSON = []byte("{}")
		}

		_, err := stmt.ExecContext(ctx,
			alert.ID,
			alert.RuleID,
			alert.RuleName,
			alert.Symbol,
			alert.Timestamp,
			alert.Price,
			alert.Message,
			string(metadataJSON),
			alert.TraceID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert alert %s: %w", alert.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close closes the database connection
func (p *AlertPersister) Close() error {
	p.Stop()
	return p.db.Close()
}

