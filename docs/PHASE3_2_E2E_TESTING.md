# Phase 3.2 E2E Testing Guide

This document provides comprehensive testing scenarios and instructions for validating the Phase 3.2 Scanner Worker Core implementation end-to-end.

## Overview

Phase 3.2 implements the complete scanner worker core, including:
- Symbol state management
- Tick ingestion
- Indicator ingestion
- Bar finalization handling
- Scan loop with rule evaluation
- Cooldown management
- Alert emission
- Partitioning & ownership
- State rehydration

## Test Structure

The E2E tests are located in `tests/scanner_e2e_test.go` and cover:

1. **Complete Flow Test** - Full workflow from rehydration to alert emission
2. **Partitioning Test** - Symbol distribution across workers
3. **Multiple Rules Test** - Complex rule evaluation scenarios

## Running the Tests

### Prerequisites

- Go 1.24+ installed
- All dependencies installed (`go mod download`)

### Run All E2E Tests

```bash
go test ./tests/... -v -run "TestScannerE2E"
```

### Run Specific Test

```bash
# Complete flow test
go test ./tests/... -v -run "TestScannerE2E_CompleteFlow"

# Partitioning test
go test ./tests/... -v -run "TestScannerE2E_Partitioning"

# Multiple rules test
go test ./tests/... -v -run "TestScannerE2E_MultipleRules"
```

### Run with Coverage

```bash
go test ./tests/... -cover -run "TestScannerE2E"
```

## Test Scenarios

### Scenario 1: Complete Flow Test

**Objective**: Validate the complete scanner workflow from state initialization to alert emission.

**Steps**:

1. **State Rehydration**
   - Load historical bars from TimescaleDB (mock)
   - Load indicators from Redis (mock)
   - Verify state is initialized correctly

2. **Rule Setup**
   - Add a rule: "RSI Oversold" (RSI < 30)
   - Reload rules in scan loop
   - Verify rule compilation

3. **Tick Ingestion**
   - Simulate tick updates for multiple symbols
   - Update live bars in state manager
   - Verify live bar state

4. **Indicator Updates**
   - Update indicator values for symbols
   - Verify indicators are stored correctly

5. **Bar Finalization**
   - Simulate finalized bar update
   - Verify bar is added to state

6. **Scan Loop Execution**
   - Run scan loop once
   - Evaluate rules against symbol state
   - Verify rule matching logic

7. **Alert Emission**
   - Verify alerts are emitted for matching rules
   - Check alert content (symbol, rule ID, price, etc.)
   - Verify alert metadata

8. **Cooldown Verification**
   - Run scan loop again immediately
   - Verify no duplicate alerts (cooldown active)
   - Verify cooldown tracking

9. **Statistics Verification**
   - Check scan loop statistics
   - Verify symbols scanned, rules evaluated, alerts emitted

**Expected Results**:
- ✅ State rehydrated with 2 symbols (AAPL, GOOGL)
- ✅ 1 alert emitted for AAPL (RSI < 30)
- ✅ No alert for GOOGL (RSI > 70)
- ✅ Cooldown prevents duplicate alerts
- ✅ Statistics tracked correctly

### Scenario 2: Partitioning Test

**Objective**: Validate symbol partitioning across multiple workers.

**Steps**:

1. **Create Partition Manager**
   - Initialize with worker ID 1, total workers 4
   - Verify initialization

2. **Test Ownership**
   - Check ownership for multiple symbols
   - Verify consistent hashing
   - Track assigned symbols

3. **Distribution Analysis**
   - Calculate partition distribution
   - Verify all symbols are distributed
   - Check partition balance

**Expected Results**:
- ✅ Symbols consistently assigned to partitions
- ✅ Worker 1 owns subset of symbols
- ✅ Distribution is balanced across partitions
- ✅ Assigned symbols tracked correctly

### Scenario 3: Multiple Rules Test

**Objective**: Validate multiple rules with different conditions.

**Steps**:

1. **Setup Multiple Rules**
   - Rule 1: RSI Oversold (< 30)
   - Rule 2: RSI Overbought (> 70)
   - Rule 3: Complex rule (RSI < 30 AND price_change_5m > 1%)

2. **Setup Symbol State**
   - Add finalized bars for price change calculation
   - Set indicators (RSI = 25.0)
   - Add live bar with price change

3. **Execute Scan**
   - Run scan loop
   - Evaluate all rules

4. **Verify Results**
   - Check which rules matched
   - Verify alert emission
   - Verify rule IDs in alerts

**Expected Results**:
- ✅ Oversold rule matches (RSI = 25 < 30)
- ✅ Overbought rule doesn't match (RSI = 25 < 70)
- ✅ Complex rule may or may not match (depends on price change)
- ✅ Correct alerts emitted

## Manual Testing Scenarios

### Scenario 4: State Rehydration with Real Data

**Objective**: Test state rehydration with actual TimescaleDB and Redis.

**Prerequisites**:
- Docker services running (`docker-compose up`)
- TimescaleDB populated with historical bars
- Redis populated with indicators

**Steps**:

1. **Prepare Data**
   ```sql
   -- Insert test bars into TimescaleDB
   INSERT INTO bars_1m (symbol, timestamp, open, high, low, close, volume, vwap)
   VALUES 
     ('AAPL', NOW() - INTERVAL '30 minutes', 150.0, 152.0, 149.0, 151.0, 1000, 150.5),
     ('AAPL', NOW() - INTERVAL '25 minutes', 151.0, 153.0, 150.0, 152.0, 1200, 151.5);
   ```

2. **Set Indicators in Redis**
   ```bash
   redis-cli SET "ind:AAPL" '{"symbol":"AAPL","timestamp":"2024-01-01T00:00:00Z","values":{"rsi_14":25.0,"ema_20":150.2}}'
   ```

3. **Run Rehydration**
   ```go
   config := scanner.DefaultRehydrationConfig()
   config.Symbols = []string{"AAPL"}
   rehydrator := scanner.NewRehydrator(config, stateManager, barStorage, redis)
   err := rehydrator.RehydrateState(context.Background())
   ```

4. **Verify State**
   - Check state manager has AAPL state
   - Verify bars are loaded
   - Verify indicators are loaded

**Expected Results**:
- ✅ State rehydrated successfully
- ✅ Bars loaded from TimescaleDB
- ✅ Indicators loaded from Redis
- ✅ State ready for scanning

### Scenario 5: Real-Time Tick Processing

**Objective**: Test real-time tick ingestion from Redis streams.

**Prerequisites**:
- Redis running
- Tick stream populated

**Steps**:

1. **Publish Ticks to Stream**
   ```bash
   redis-cli XADD ticks:0 * tick '{"symbol":"AAPL","price":152.5,"size":100,"timestamp":"2024-01-01T12:00:00Z","type":"trade"}'
   ```

2. **Start Tick Consumer**
   ```go
   config := pubsub.DefaultStreamConsumerConfig("ticks", "scanner-group", "scanner-1")
   tickConsumer := scanner.NewTickConsumer(redis, config, stateManager)
   tickConsumer.Start()
   ```

3. **Monitor State Updates**
   - Check live bar updates
   - Verify tick processing stats

4. **Stop Consumer**
   ```go
   tickConsumer.Stop()
   ```

**Expected Results**:
- ✅ Ticks consumed from stream
- ✅ Live bars updated in state
- ✅ Statistics tracked correctly
- ✅ Graceful shutdown

### Scenario 6: Rule Evaluation with Real Indicators

**Objective**: Test rule evaluation with actual indicator values.

**Steps**:

1. **Setup Rule**
   ```json
   {
     "id": "rule-oversold",
     "name": "RSI Oversold",
     "conditions": [
       {"metric": "rsi_14", "op": "<", "value": 30.0}
     ],
     "cooldown_sec": 10,
     "enabled": true
   }
   ```

2. **Set Indicators**
   ```bash
   redis-cli SET "ind:AAPL" '{"symbol":"AAPL","values":{"rsi_14":25.0}}'
   ```

3. **Update Indicators in State**
   ```go
   stateManager.UpdateIndicators("AAPL", map[string]float64{"rsi_14": 25.0})
   ```

4. **Run Scan Loop**
   ```go
   scanLoop.scan()
   ```

5. **Verify Alert**
   - Check alert emitter received alert
   - Verify alert content

**Expected Results**:
- ✅ Rule matches (RSI 25 < 30)
- ✅ Alert emitted
- ✅ Alert contains correct data

### Scenario 7: Cooldown Enforcement

**Objective**: Verify cooldown prevents duplicate alerts.

**Steps**:

1. **Setup Rule with Cooldown**
   - Cooldown: 10 seconds

2. **First Scan**
   - Rule matches
   - Alert emitted
   - Cooldown recorded

3. **Immediate Second Scan**
   - Rule still matches
   - No alert emitted (cooldown active)

4. **Wait for Cooldown**
   - Wait 11 seconds
   - Run scan again
   - Alert emitted (cooldown expired)

**Expected Results**:
- ✅ First alert emitted
- ✅ Second scan: no alert (cooldown)
- ✅ After cooldown: alert emitted again

### Scenario 8: Partitioning with Multiple Workers

**Objective**: Test symbol distribution across multiple workers.

**Steps**:

1. **Create Multiple Partition Managers**
   ```go
   pm1, _ := scanner.NewPartitionManager(0, 4)
   pm2, _ := scanner.NewPartitionManager(1, 4)
   pm3, _ := scanner.NewPartitionManager(2, 4)
   pm4, _ := scanner.NewPartitionManager(3, 4)
   ```

2. **Test Symbol Distribution**
   ```go
   symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN", "NVDA", "META", "NFLX"}
   for _, symbol := range symbols {
       partition := pm1.GetPartition(symbol)
       // Verify symbol goes to correct worker
   }
   ```

3. **Verify Balance**
   - Check distribution across workers
   - Verify no symbol assigned to multiple workers

**Expected Results**:
- ✅ Symbols distributed across workers
- ✅ Each symbol assigned to exactly one worker
- ✅ Distribution is balanced

## Performance Testing

### Scenario 9: Load Test with Many Symbols

**Objective**: Test scanner performance with large number of symbols.

**Steps**:

1. **Setup 1000+ Symbols**
   ```go
   for i := 0; i < 1000; i++ {
       symbol := fmt.Sprintf("SYMBOL%d", i)
       // Setup state for each symbol
   }
   ```

2. **Run Scan Loop**
   ```go
   start := time.Now()
   scanLoop.scan()
   duration := time.Since(start)
   ```

3. **Verify Performance**
   - Scan time < 800ms (target)
   - Check statistics
   - Monitor memory usage

**Expected Results**:
- ✅ Scan completes in < 800ms
- ✅ All symbols scanned
- ✅ Rules evaluated correctly
- ✅ Memory usage reasonable

### Scenario 10: Concurrent Operations

**Objective**: Test thread safety with concurrent operations.

**Steps**:

1. **Concurrent State Updates**
   ```go
   // Multiple goroutines updating state
   for i := 0; i < 10; i++ {
       go func() {
           stateManager.UpdateLiveBar("AAPL", tick)
           stateManager.UpdateIndicators("AAPL", indicators)
       }()
   }
   ```

2. **Concurrent Scan Loop**
   ```go
   // Run scan loop while updating state
   go scanLoop.scan()
   go stateManager.UpdateLiveBar("AAPL", tick)
   ```

3. **Verify No Race Conditions**
   - Run with race detector: `go test -race`
   - Check for panics
   - Verify data consistency

**Expected Results**:
- ✅ No race conditions detected
- ✅ No panics
- ✅ Data remains consistent

## Integration Testing

### Scenario 11: Full Pipeline Integration

**Objective**: Test complete pipeline from ingestion to alert emission.

**Prerequisites**:
- All services running (ingest, bars, indicator, scanner)

**Steps**:

1. **Start Ingest Service**
   - Publishes ticks to Redis stream

2. **Start Bars Service**
   - Consumes ticks, aggregates bars
   - Publishes finalized bars

3. **Start Indicator Service**
   - Consumes finalized bars
   - Computes indicators
   - Publishes indicators

4. **Start Scanner Service**
   - Rehydrates state
   - Consumes ticks, bars, indicators
   - Runs scan loop
   - Emits alerts

5. **Monitor Alerts**
   - Check Redis pub/sub for alerts
   - Verify alert content

**Expected Results**:
- ✅ All services running
- ✅ Data flows through pipeline
- ✅ Alerts emitted correctly
- ✅ No data loss

## Troubleshooting

### Common Issues

1. **State Not Rehydrating**
   - Check TimescaleDB connection
   - Verify bars exist in database
   - Check Redis connection
   - Verify indicator keys exist

2. **Rules Not Matching**
   - Verify rule conditions
   - Check indicator values
   - Verify metrics are computed correctly
   - Check rule compilation

3. **No Alerts Emitted**
   - Check if rules match
   - Verify cooldown status
   - Check alert emitter configuration
   - Verify Redis connection

4. **Performance Issues**
   - Check scan cycle time
   - Monitor memory usage
   - Verify symbol count
   - Check rule count

### Debug Commands

```bash
# Run tests with verbose output
go test ./tests/... -v -run "TestScannerE2E"

# Run with race detector
go test ./tests/... -race -run "TestScannerE2E"

# Run with coverage
go test ./tests/... -cover -run "TestScannerE2E"

# Run specific test
go test ./tests/... -v -run "TestScannerE2E_CompleteFlow"
```

## Success Criteria

All tests should pass with:
- ✅ No test failures
- ✅ No race conditions
- ✅ Code coverage > 60%
- ✅ All scenarios validated
- ✅ Performance targets met (< 800ms scan time)

## Next Steps

After validating Phase 3.2:
1. Proceed to Phase 3.3 (Scanner Worker Service)
2. Integrate with main service
3. Deploy to staging environment
4. Monitor in production

