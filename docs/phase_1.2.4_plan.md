# Phase 1.2.4: TimescaleDB Integration - Implementation Plan

## Overview
Implement TimescaleDB writer to persist finalized bars to the database. This will enable historical data storage and retrieval for the bar aggregator service.

## Requirements

### Database Schema
- Table: `bars_1m` (already created in Phase 0)
- Columns: `symbol`, `timestamp`, `open`, `high`, `low`, `close`, `volume`, `vwap`
- Primary Key: `(symbol, timestamp)`
- Hypertable: Partitioned by `timestamp`
- Indexes: On `symbol`, `timestamp`, and composite `(symbol, timestamp)`

### Functional Requirements
1. **Connection Management**
   - Connection pooling (configurable max connections)
   - Connection lifecycle management
   - Health checks

2. **Write Operations**
   - Batch insert for finalized bars (efficient bulk operations)
   - Async write queue (non-blocking)
   - Automatic retry on failures
   - Error handling and logging

3. **Read Operations** (for future use)
   - Get bars by symbol and time range
   - Get latest N bars for a symbol

4. **Observability**
   - Metrics for write latency
   - Metrics for write errors
   - Metrics for queue depth

## Implementation Steps

### Step 1: Create TimescaleDB Client (`internal/storage/timescale.go`)

**File Structure:**
```go
package storage

import (
    "context"
    "database/sql"
    "time"
    "github.com/lib/pq" // PostgreSQL driver
    "github.com/mohamedkhairy/stock-scanner/internal/config"
    "github.com/mohamedkhairy/stock-scanner/internal/models"
)

type TimescaleDBClient struct {
    db     *sql.DB
    config config.DatabaseConfig
    // Write queue
    writeQueue chan []*models.Bar1m
    // Metrics
    // ...
}
```

**Key Components:**
1. **Connection Pool Setup**
   - Use `database/sql` with `lib/pq` driver
   - Configure max connections, idle connections, connection lifetime
   - Connection string: `postgres://user:password@host:port/database?sslmode=...`

2. **Write Queue**
   - Buffered channel for async writes
   - Background goroutine to process queue
   - Batch accumulation (collect bars until batch size or timeout)

3. **Batch Insert**
   - Use PostgreSQL `COPY` or batch `INSERT` with `VALUES`
   - Handle conflicts (ON CONFLICT DO NOTHING or UPDATE)
   - Transaction support for atomicity

4. **Error Handling**
   - Retry logic with exponential backoff
   - Dead letter queue for failed writes (optional)
   - Logging and metrics

### Step 2: Implement BarStorage Interface

**Methods to Implement:**
```go
// WriteBars writes finalized bars to storage
WriteBars(ctx context.Context, bars []*models.Bar1m) error

// GetBars retrieves bars for a symbol within a time range
GetBars(ctx context.Context, symbol string, start, end time.Time) ([]*models.Bar1m, error)

// GetLatestBars retrieves the latest N bars for a symbol
GetLatestBars(ctx context.Context, symbol string, limit int) ([]*models.Bar1m, error)

// Close closes the storage connection
Close() error
```

**Implementation Details:**
- `WriteBars`: Enqueue bars to write queue (non-blocking)
- `GetBars`: Query with WHERE clause on symbol and timestamp range
- `GetLatestBars`: Query with ORDER BY timestamp DESC LIMIT
- `Close`: Gracefully shutdown, flush queue, close connections

### Step 3: Batch Processing Logic

**Queue Processor:**
- Collect bars from queue
- Batch when:
  - Batch size reached (e.g., 1000 bars)
  - Timeout reached (e.g., 1 second)
- Execute batch insert
- Handle errors and retries

**SQL Query:**
```sql
INSERT INTO bars_1m (symbol, timestamp, open, high, low, close, volume, vwap)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (symbol, timestamp) DO UPDATE SET
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume,
    vwap = EXCLUDED.vwap;
```

Or use batch insert:
```sql
INSERT INTO bars_1m (symbol, timestamp, open, high, low, close, volume, vwap)
VALUES 
    ($1, $2, $3, $4, $5, $6, $7, $8),
    ($9, $10, $11, $12, $13, $14, $15, $16),
    ...
ON CONFLICT (symbol, timestamp) DO UPDATE SET ...
```

### Step 4: Add Metrics

**Prometheus Metrics:**
- `timescale_write_total` - Counter of write operations
- `timescale_write_errors_total` - Counter of write errors
- `timescale_write_latency_seconds` - Histogram of write latency
- `timescale_write_queue_depth` - Gauge of queue depth
- `timescale_write_batch_size` - Histogram of batch sizes

### Step 5: Configuration

**Add to `BarsConfig`:**
```go
type BarsConfig struct {
    // ... existing fields
    DBWriteBatchSize   int           // Batch size for DB writes
    DBWriteInterval    time.Duration // Interval for flushing batches
    DBWriteQueueSize   int           // Size of write queue
    DBMaxRetries       int           // Max retries for failed writes
    DBRetryDelay       time.Duration // Initial retry delay
}
```

### Step 6: Integration with Bar Publisher

**Update `internal/bars/publisher.go`:**
- Add `BarStorage` field
- Call `WriteBars()` when bars are finalized
- Handle write errors gracefully (log but don't block)

### Step 7: Testing

**Unit Tests:**
- Test connection setup
- Test batch insert
- Test error handling and retries
- Test queue processing
- Test read operations

**Integration Tests:**
- Test with real TimescaleDB (Docker)
- Test batch writes
- Test concurrent writes
- Test queue overflow handling

## File Structure

```
internal/storage/
  ├── timescale.go          # TimescaleDB client implementation
  ├── timescale_test.go     # Unit tests
  └── interfaces.go         # Already exists (BarStorage interface)

internal/bars/
  └── publisher.go          # Update to integrate with TimescaleDB

internal/config/
  └── config.go             # Update BarsConfig

scripts/migrations/
  └── 001_create_bars_table.sql  # Already exists
```

## Dependencies

**New Dependencies:**
- `github.com/lib/pq` - PostgreSQL driver for Go
- `github.com/prometheus/client_golang` - Already in use for metrics

**Add to `go.mod`:**
```bash
go get github.com/lib/pq
```

## Configuration Example

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=stock_scanner
DB_SSL_MODE=disable
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Bar Aggregator DB Write Configuration
BARS_DB_WRITE_BATCH_SIZE=1000
BARS_DB_WRITE_INTERVAL=1s
BARS_DB_WRITE_QUEUE_SIZE=10000
BARS_DB_MAX_RETRIES=3
BARS_DB_RETRY_DELAY=100ms
```

## Error Handling Strategy

1. **Transient Errors** (network, timeout)
   - Retry with exponential backoff
   - Max retries: 3
   - Log warnings

2. **Permanent Errors** (invalid data, constraint violations)
   - Log errors
   - Skip invalid bars
   - Continue processing

3. **Queue Full**
   - Log warning
   - Optionally drop oldest or reject new writes
   - Monitor queue depth

## Performance Considerations

1. **Batch Size**: 1000 bars per batch (configurable)
2. **Queue Size**: 10,000 bars (configurable)
3. **Connection Pool**: 25 max connections (configurable)
4. **Write Interval**: 1 second (configurable)
5. **Use Transactions**: For atomic batch writes

## Success Criteria

- ✅ TimescaleDB client implements `BarStorage` interface
- ✅ Connection pooling configured correctly
- ✅ Batch inserts working efficiently
- ✅ Async write queue processing bars
- ✅ Error handling and retries working
- ✅ Metrics exposed and working
- ✅ Unit tests passing
- ✅ Integration tests passing
- ✅ No data loss during writes
- ✅ Graceful shutdown flushes queue

## Next Steps After Completion

1. Phase 1.2.5: Bar Aggregator Service Main
   - Wire everything together
   - Create `cmd/bars/main.go`
   - Integration with consumer, aggregator, publisher, and TimescaleDB

2. Phase 1.3: Testing & Validation
   - End-to-end test: Ingest → Bars → TimescaleDB
   - Load testing
   - Verify data accuracy

