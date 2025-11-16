package pubsub

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Metrics for stream publishing
	publishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stream_publish_total",
			Help: "Total number of messages published to streams",
		},
		[]string{"stream", "partition"},
	)

	publishErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stream_publish_errors_total",
			Help: "Total number of publish errors",
		},
		[]string{"stream", "partition"},
	)

	publishLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stream_publish_latency_seconds",
			Help:    "Publish latency in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"stream", "partition"},
	)

	batchSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stream_publish_batch_size",
			Help:    "Batch size for stream publishing",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"stream"},
	)
)

// StreamPublisherConfig holds configuration for the stream publisher
type StreamPublisherConfig struct {
	StreamName    string
	BatchSize     int
	BatchTimeout  time.Duration
	Partitions    int // Number of partitions (0 = no partitioning)
	RetryAttempts int
	RetryDelay    time.Duration
}

// DefaultStreamPublisherConfig returns default configuration
func DefaultStreamPublisherConfig(streamName string) StreamPublisherConfig {
	return StreamPublisherConfig{
		StreamName:    streamName,
		BatchSize:     100,
		BatchTimeout:  100 * time.Millisecond,
		Partitions:    0, // No partitioning by default
		RetryAttempts: 3,
		RetryDelay:    100 * time.Millisecond,
	}
}

// StreamPublisher publishes ticks to Redis streams with batching and partitioning
type StreamPublisher struct {
	config     StreamPublisherConfig
	redis      storage.RedisClient
	batch      []*models.Tick
	batchMu    sync.Mutex
	ticker     *time.Ticker
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewStreamPublisher creates a new stream publisher
func NewStreamPublisher(redis storage.RedisClient, config StreamPublisherConfig) *StreamPublisher {
	ctx, cancel := context.WithCancel(context.Background())

	return &StreamPublisher{
		config: config,
		redis:  redis,
		batch:  make([]*models.Tick, 0, config.BatchSize),
		ticker: time.NewTicker(config.BatchTimeout),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts the batch publishing loop
func (p *StreamPublisher) Start() {
	p.wg.Add(1)
	go p.batchLoop()
}

// Publish adds a tick to the batch (non-blocking)
func (p *StreamPublisher) Publish(tick *models.Tick) error {
	if tick == nil {
		return fmt.Errorf("tick cannot be nil")
	}

	if err := tick.Validate(); err != nil {
		return fmt.Errorf("invalid tick: %w", err)
	}

	p.batchMu.Lock()
	p.batch = append(p.batch, tick)
	shouldFlush := len(p.batch) >= p.config.BatchSize
	p.batchMu.Unlock()

	// Flush if batch is full
	if shouldFlush {
		return p.flush()
	}

	return nil
}

// batchLoop periodically flushes the batch
func (p *StreamPublisher) batchLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			// Flush remaining items on shutdown
			p.flush()
			return
		case <-p.ticker.C:
			p.flush()
		}
	}
}

// flush publishes the current batch to Redis streams
func (p *StreamPublisher) flush() error {
	p.batchMu.Lock()
	if len(p.batch) == 0 {
		p.batchMu.Unlock()
		return nil
	}

	// Copy batch and clear
	batch := make([]*models.Tick, len(p.batch))
	copy(batch, p.batch)
	p.batch = p.batch[:0]
	p.batchMu.Unlock()

	// Record batch size metric
	batchSize.WithLabelValues(p.config.StreamName).Observe(float64(len(batch)))

	// Group ticks by partition (if partitioning is enabled)
	if p.config.Partitions > 0 {
		return p.publishPartitioned(batch)
	}

	// Publish all to single stream
	return p.publishBatch(batch, p.config.StreamName, "")
}

// publishPartitioned publishes ticks to partitioned streams
func (p *StreamPublisher) publishPartitioned(ticks []*models.Tick) error {
	// Group ticks by partition
	partitions := make(map[int][]*models.Tick)

	for _, tick := range ticks {
		partition := p.getPartition(tick.Symbol)
		partitions[partition] = append(partitions[partition], tick)
	}

	// Publish each partition
	var lastErr error
	for partition, partitionTicks := range partitions {
		streamName := fmt.Sprintf("%s.p%d", p.config.StreamName, partition)
		err := p.publishBatch(partitionTicks, streamName, fmt.Sprintf("%d", partition))
		if err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// publishBatch publishes a batch of ticks to a stream using individual messages
func (p *StreamPublisher) publishBatch(ticks []*models.Tick, streamName string, partition string) error {
	startTime := time.Now()

	if len(ticks) == 0 {
		return nil
	}

	// Serialize all ticks and prepare batch messages
	messages := make([]map[string]interface{}, 0, len(ticks))
	for _, tick := range ticks {
		tickJSON, marshalErr := json.Marshal(tick)
		if marshalErr != nil {
			logger.Error("Failed to marshal tick",
				logger.ErrorField(marshalErr),
				logger.String("symbol", tick.Symbol),
			)
			continue
		}
		messages = append(messages, map[string]interface{}{
			"tick": string(tickJSON),
		})
	}

	if len(messages) == 0 {
		return nil
	}

	// Publish batch using pipeline with retries
	var err error
	for attempt := 0; attempt < p.config.RetryAttempts; attempt++ {
		err = p.redis.PublishBatchToStream(p.ctx, streamName, messages)
		if err == nil {
			break
		}

		if attempt < p.config.RetryAttempts-1 {
			logger.Warn("Failed to publish batch, retrying",
				logger.ErrorField(err),
				logger.String("stream", streamName),
				logger.Int("attempt", attempt+1),
				logger.Int("count", len(messages)),
			)
			time.Sleep(p.config.RetryDelay * time.Duration(attempt+1))
		}
	}

	latency := time.Since(startTime).Seconds()

	if err != nil {
		publishErrors.WithLabelValues(streamName, partition).Add(float64(len(messages)))
		logger.Error("Failed to publish batch after retries",
			logger.ErrorField(err),
			logger.String("stream", streamName),
			logger.Int("count", len(messages)),
		)
		return err
	}

	// Record metrics
	publishTotal.WithLabelValues(streamName, partition).Add(float64(len(messages)))
	publishLatency.WithLabelValues(streamName, partition).Observe(latency)

	logger.Debug("Published batch to stream",
		logger.String("stream", streamName),
		logger.Int("count", len(messages)),
		logger.Duration("latency", time.Since(startTime)),
	)

	return nil
}

// getPartition calculates the partition for a symbol using hash-based partitioning
func (p *StreamPublisher) getPartition(symbol string) int {
	if p.config.Partitions == 0 {
		return 0
	}

	hash := sha256.Sum256([]byte(symbol))
	hashInt := int(hash[0])<<24 | int(hash[1])<<16 | int(hash[2])<<8 | int(hash[3])
	if hashInt < 0 {
		hashInt = -hashInt
	}
	return hashInt % p.config.Partitions
}

// GetPartitionStreamName returns the stream name for a given partition
func (p *StreamPublisher) GetPartitionStreamName(partition int) string {
	if p.config.Partitions == 0 {
		return p.config.StreamName
	}
	return fmt.Sprintf("%s.p%d", p.config.StreamName, partition)
}

// Flush forces an immediate flush of the current batch
func (p *StreamPublisher) Flush() error {
	return p.flush()
}

// Close stops the publisher and flushes remaining items
func (p *StreamPublisher) Close() error {
	p.cancel()
	p.ticker.Stop()
	p.wg.Wait()
	return p.flush()
}

// GetBatchSize returns the current batch size (for monitoring)
func (p *StreamPublisher) GetBatchSize() int {
	p.batchMu.Lock()
	defer p.batchMu.Unlock()
	return len(p.batch)
}

