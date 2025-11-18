package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Metrics for TimescaleDB operations
	timescaleWriteTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "timescale_write_total",
			Help: "Total number of write operations to TimescaleDB",
		},
		[]string{"status"}, // "success" or "error"
	)

	timescaleWriteErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "timescale_write_errors_total",
			Help: "Total number of write errors to TimescaleDB",
		},
		[]string{"error_type"},
	)

	timescaleWriteLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "timescale_write_latency_seconds",
			Help:    "Write latency to TimescaleDB in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0},
		},
		[]string{"operation"},
	)

	timescaleWriteQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "timescale_write_queue_depth",
			Help: "Current depth of the write queue",
		},
	)

	timescaleWriteBatchSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "timescale_write_batch_size",
			Help:    "Batch size for TimescaleDB writes",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"operation"},
	)
)

// TimescaleDBClient implements BarStorage interface for TimescaleDB
type TimescaleDBClient struct {
	db          *sql.DB
	dbConfig    config.DatabaseConfig
	writeConfig WriteConfig

	// Write queue
	writeQueue chan []*models.Bar1m
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

// WriteConfigFromBarsConfig creates a WriteConfig from BarsConfig
func WriteConfigFromBarsConfig(barsConfig config.BarsConfig) WriteConfig {
	return WriteConfig{
		BatchSize:  barsConfig.DBWriteBatchSize,
		Interval:   barsConfig.DBWriteInterval,
		QueueSize:  barsConfig.DBWriteQueueSize,
		MaxRetries: barsConfig.DBMaxRetries,
		RetryDelay: barsConfig.DBRetryDelay,
	}
}

// NewTimescaleDBClient creates a new TimescaleDB client
func NewTimescaleDBClient(dbConfig config.DatabaseConfig, writeConfig WriteConfig) (*TimescaleDBClient, error) {
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

	client := &TimescaleDBClient{
		db:          db,
		dbConfig:    dbConfig,
		writeConfig: writeConfig,
		writeQueue:  make(chan []*models.Bar1m, writeConfig.QueueSize),
		ctx:         clientCtx,
		cancel:      clientCancel,
	}

	logger.Info("Connected to TimescaleDB",
		logger.String("host", dbConfig.Host),
		logger.Int("port", dbConfig.Port),
		logger.String("database", dbConfig.Database),
	)

	return client, nil
}

// Start starts the write queue processor
func (t *TimescaleDBClient) Start() error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return fmt.Errorf("TimescaleDB client is already running")
	}
	t.running = true
	t.mu.Unlock()

	logger.Info("Starting TimescaleDB write queue processor",
		logger.Int("batch_size", t.writeConfig.BatchSize),
		logger.Duration("interval", t.writeConfig.Interval),
	)

	t.wg.Add(1)
	go t.processWriteQueue()

	return nil
}

// Stop stops the write queue processor and flushes remaining writes
func (t *TimescaleDBClient) Stop() error {
	t.mu.Lock()
	if !t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = false
	t.mu.Unlock()

	logger.Info("Stopping TimescaleDB write queue processor")
	t.cancel()

	// Flush remaining writes
	close(t.writeQueue)
	for bars := range t.writeQueue {
		if len(bars) > 0 {
			t.writeBarsSync(context.Background(), bars)
		}
	}

	t.wg.Wait()

	// Close database connection
	if err := t.db.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	logger.Info("TimescaleDB client stopped")
	return nil
}

// WriteBars enqueues bars for async writing
func (t *TimescaleDBClient) WriteBars(ctx context.Context, bars []*models.Bar1m) error {
	if len(bars) == 0 {
		return nil
	}

	// Validate bars
	validBars := make([]*models.Bar1m, 0, len(bars))
	for _, bar := range bars {
		if err := bar.Validate(); err != nil {
			logger.Warn("Invalid bar, skipping",
				logger.ErrorField(err),
				logger.String("symbol", bar.Symbol),
			)
			continue
		}
		validBars = append(validBars, bar)
	}

	if len(validBars) == 0 {
		return nil
	}

	// Try to enqueue (non-blocking with timeout)
	select {
	case t.writeQueue <- validBars:
		timescaleWriteQueueDepth.Set(float64(len(t.writeQueue)))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
		// Queue might be full, log warning but still try
		logger.Warn("Write queue may be full, attempting to enqueue",
			logger.Int("queue_depth", len(t.writeQueue)),
			logger.Int("bars_count", len(validBars)),
		)
		select {
		case t.writeQueue <- validBars:
			timescaleWriteQueueDepth.Set(float64(len(t.writeQueue)))
			return nil
		default:
			timescaleWriteErrors.WithLabelValues("queue_full").Inc()
			return fmt.Errorf("write queue is full")
		}
	}
}

// GetBars retrieves bars for a symbol within a time range
func (t *TimescaleDBClient) GetBars(ctx context.Context, symbol string, start, end time.Time) ([]*models.Bar1m, error) {
	query := `
		SELECT symbol, timestamp, open, high, low, close, volume, vwap
		FROM bars_1m
		WHERE symbol = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp ASC
	`

	rows, err := t.db.QueryContext(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query bars: %w", err)
	}
	defer rows.Close()

	var bars []*models.Bar1m
	for rows.Next() {
		var bar models.Bar1m
		if err := rows.Scan(
			&bar.Symbol,
			&bar.Timestamp,
			&bar.Open,
			&bar.High,
			&bar.Low,
			&bar.Close,
			&bar.Volume,
			&bar.VWAP,
		); err != nil {
			return nil, fmt.Errorf("failed to scan bar: %w", err)
		}
		bars = append(bars, &bar)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return bars, nil
}

// GetLatestBars retrieves the latest N bars for a symbol
func (t *TimescaleDBClient) GetLatestBars(ctx context.Context, symbol string, limit int) ([]*models.Bar1m, error) {
	query := `
		SELECT symbol, timestamp, open, high, low, close, volume, vwap
		FROM bars_1m
		WHERE symbol = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := t.db.QueryContext(ctx, query, symbol, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest bars: %w", err)
	}
	defer rows.Close()

	var bars []*models.Bar1m
	for rows.Next() {
		var bar models.Bar1m
		if err := rows.Scan(
			&bar.Symbol,
			&bar.Timestamp,
			&bar.Open,
			&bar.High,
			&bar.Low,
			&bar.Close,
			&bar.Volume,
			&bar.VWAP,
		); err != nil {
			return nil, fmt.Errorf("failed to scan bar: %w", err)
		}
		bars = append(bars, &bar)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Reverse to get chronological order
	for i, j := 0, len(bars)-1; i < j; i, j = i+1, j-1 {
		bars[i], bars[j] = bars[j], bars[i]
	}

	return bars, nil
}

// Close closes the database connection
func (t *TimescaleDBClient) Close() error {
	return t.Stop()
}

// processWriteQueue processes the write queue
func (t *TimescaleDBClient) processWriteQueue() {
	defer t.wg.Done()

	batch := make([]*models.Bar1m, 0, t.writeConfig.BatchSize)
	ticker := time.NewTicker(t.writeConfig.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			// Flush remaining batch
			if len(batch) > 0 {
				t.writeBarsSync(context.Background(), batch)
			}
			return

		case bars, ok := <-t.writeQueue:
			if !ok {
				// Channel closed, flush and exit
				if len(batch) > 0 {
					t.writeBarsSync(context.Background(), batch)
				}
				return
			}

			batch = append(batch, bars...)
			timescaleWriteQueueDepth.Set(float64(len(t.writeQueue)))

			// Flush if batch is full
			if len(batch) >= t.writeConfig.BatchSize {
				t.writeBarsSync(context.Background(), batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			// Flush on interval
			if len(batch) > 0 {
				t.writeBarsSync(context.Background(), batch)
				batch = batch[:0]
			}
		}
	}
}

// writeBarsSync writes bars synchronously with retry logic
func (t *TimescaleDBClient) writeBarsSync(ctx context.Context, bars []*models.Bar1m) {
	if len(bars) == 0 {
		return
	}

	startTime := time.Now()
	timescaleWriteBatchSize.WithLabelValues("write").Observe(float64(len(bars)))

	var err error
	for attempt := 0; attempt < t.writeConfig.MaxRetries; attempt++ {
		err = t.insertBars(ctx, bars)
		if err == nil {
			break
		}

		if attempt < t.writeConfig.MaxRetries-1 {
			delay := t.writeConfig.RetryDelay * time.Duration(1<<uint(attempt)) // Exponential backoff
			logger.Warn("Failed to write bars, retrying",
				logger.ErrorField(err),
				logger.Int("attempt", attempt+1),
				logger.Int("bars_count", len(bars)),
				logger.Duration("delay", delay),
			)
			time.Sleep(delay)
		}
	}

	latency := time.Since(startTime).Seconds()
	timescaleWriteLatency.WithLabelValues("write").Observe(latency)

	if err != nil {
		timescaleWriteErrors.WithLabelValues("write_failed").Inc()
		timescaleWriteTotal.WithLabelValues("error").Add(float64(len(bars)))
		logger.Error("Failed to write bars after retries",
			logger.ErrorField(err),
			logger.Int("bars_count", len(bars)),
		)
		return
	}

	timescaleWriteTotal.WithLabelValues("success").Add(float64(len(bars)))
	logger.Debug("Wrote bars to TimescaleDB",
		logger.Int("count", len(bars)),
		logger.Duration("latency", time.Since(startTime)),
	)
}

// insertBars inserts bars into the database using batch insert
func (t *TimescaleDBClient) insertBars(ctx context.Context, bars []*models.Bar1m) error {
	if len(bars) == 0 {
		return nil
	}

	// Use transaction for atomicity
	tx, err := t.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare statement for batch insert
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO bars_1m (symbol, timestamp, open, high, low, close, volume, vwap)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (symbol, timestamp) DO UPDATE SET
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			vwap = EXCLUDED.vwap
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Execute batch insert
	for _, bar := range bars {
		_, err := stmt.ExecContext(ctx,
			bar.Symbol,
			bar.Timestamp,
			bar.Open,
			bar.High,
			bar.Low,
			bar.Close,
			bar.Volume,
			bar.VWAP,
		)
		if err != nil {
			return fmt.Errorf("failed to insert bar: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// IsRunning returns whether the client is running
func (t *TimescaleDBClient) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}

