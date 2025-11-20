// Package data contains load tests for the stock scanner system.
//
// These tests verify system performance under various symbol counts:
// - 2000 symbols
// - 5000 symbols
// - 10000 symbols
// - Tick ingestion rates
// - Concurrent state updates
//
// See README.md for documentation on running performance tests.
package data

import (
	"fmt"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/scanner"
)

// TestLoad_2000Symbols tests the system with 2000 symbols
func TestLoad_2000Symbols(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	symbolCount := 2000

	// Generate symbols
	symbols := generateSymbols(symbolCount)

	// Setup state manager
	sm := scanner.NewStateManager(200)

	// Measure scan time
	start := time.Now()
	
	// Initialize state for all symbols
	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		if err := sm.UpdateLiveBar(symbol, tick); err != nil {
			t.Fatalf("Failed to update live bar for %s: %v", symbol, err)
		}
	}

	// Simulate scan loop
	scanStart := time.Now()
	sm.Snapshot() // This simulates the snapshot operation in scan loop
	scanDuration := time.Since(scanStart)

	elapsed := time.Since(start)

	t.Logf("Load test results for %d symbols:", symbolCount)
	t.Logf("  Total setup time: %v", elapsed)
	t.Logf("  Scan snapshot time: %v", scanDuration)
	t.Logf("  Symbols per second: %.2f", float64(symbolCount)/elapsed.Seconds())

	// Verify performance target (< 800ms for scan cycle)
	if scanDuration > 800*time.Millisecond {
		t.Errorf("Scan cycle time (%v) exceeds target (800ms)", scanDuration)
	}

	// Verify all symbols are in state
	if sm.GetSymbolCount() != symbolCount {
		t.Errorf("Expected %d symbols in state, got %d", symbolCount, sm.GetSymbolCount())
	}
}

// TestLoad_5000Symbols tests the system with 5000 symbols
func TestLoad_5000Symbols(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	symbolCount := 5000

	// Generate symbols
	symbols := generateSymbols(symbolCount)

	// Setup state manager
	sm := scanner.NewStateManager(200)

	// Measure scan time
	start := time.Now()
	
	// Initialize state for all symbols
	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		if err := sm.UpdateLiveBar(symbol, tick); err != nil {
			t.Fatalf("Failed to update live bar for %s: %v", symbol, err)
		}
	}

	// Simulate scan loop
	scanStart := time.Now()
	sm.Snapshot()
	scanDuration := time.Since(scanStart)

	elapsed := time.Since(start)

	t.Logf("Load test results for %d symbols:", symbolCount)
	t.Logf("  Total setup time: %v", elapsed)
	t.Logf("  Scan snapshot time: %v", scanDuration)
	t.Logf("  Symbols per second: %.2f", float64(symbolCount)/elapsed.Seconds())

	// For 5000 symbols, we expect higher latency but still reasonable
	if scanDuration > 2*time.Second {
		t.Errorf("Scan cycle time (%v) is too high for %d symbols", scanDuration, symbolCount)
	}
}

// TestLoad_10000Symbols tests the system with 10000 symbols
func TestLoad_10000Symbols(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	symbolCount := 10000

	// Generate symbols
	symbols := generateSymbols(symbolCount)

	// Setup state manager
	sm := scanner.NewStateManager(200)

	// Measure scan time
	start := time.Now()
	
	// Initialize state for all symbols (batch updates)
	batchSize := 1000
	for i := 0; i < len(symbols); i += batchSize {
		end := i + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		
		for _, symbol := range symbols[i:end] {
			tick := &models.Tick{
				Symbol:    symbol,
				Price:     100.0,
				Size:      100,
				Timestamp: time.Now(),
				Type:      "trade",
			}
			if err := sm.UpdateLiveBar(symbol, tick); err != nil {
				t.Fatalf("Failed to update live bar for %s: %v", symbol, err)
			}
		}
	}

	// Simulate scan loop
	scanStart := time.Now()
	sm.Snapshot()
	scanDuration := time.Since(scanStart)

	elapsed := time.Since(start)

	t.Logf("Load test results for %d symbols:", symbolCount)
	t.Logf("  Total setup time: %v", elapsed)
	t.Logf("  Scan snapshot time: %v", scanDuration)
	t.Logf("  Symbols per second: %.2f", float64(symbolCount)/elapsed.Seconds())

	// For 10000 symbols, we expect higher latency
	if scanDuration > 5*time.Second {
		t.Errorf("Scan cycle time (%v) is too high for %d symbols", scanDuration, symbolCount)
	}
}

// TestLoad_TickIngestionRate tests tick ingestion rate
func TestLoad_TickIngestionRate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	// Setup Redis client
	redisClient, err := pubsub.NewRedisClient(config.RedisConfig{
		Host:         "localhost",
		Port:         6379,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
	})
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
	}
	defer redisClient.Close()

	// Setup publisher
	publisherConfig := pubsub.DefaultStreamPublisherConfig("ticks")
	publisher := pubsub.NewStreamPublisher(redisClient, publisherConfig)
	publisher.Start()
	defer publisher.Close()

	symbolCount := 100
	symbols := generateSymbols(symbolCount)
	ticksPerSymbol := 100
	totalTicks := symbolCount * ticksPerSymbol

	// Measure ingestion rate
	start := time.Now()
	
	for _, symbol := range symbols {
		for i := 0; i < ticksPerSymbol; i++ {
			tick := &models.Tick{
				Symbol:    symbol,
				Price:     100.0 + float64(i)*0.01,
				Size:      100,
				Timestamp: time.Now(),
				Type:      "trade",
			}
			if err := publisher.Publish(tick); err != nil {
				t.Fatalf("Failed to publish tick: %v", err)
			}
		}
	}
	
	publisher.Flush()

	elapsed := time.Since(start)
	rate := float64(totalTicks) / elapsed.Seconds()

	t.Logf("Tick ingestion test results:")
	t.Logf("  Total ticks: %d", totalTicks)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Rate: %.2f ticks/second", rate)

	// Verify we can ingest at least 1000 ticks/second
	if rate < 1000 {
		t.Errorf("Ingestion rate (%.2f ticks/s) is below target (1000 ticks/s)", rate)
	}
}

// TestLoad_ConcurrentUpdates tests concurrent state updates
func TestLoad_ConcurrentUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	symbolCount := 1000
	symbols := generateSymbols(symbolCount)
	concurrency := 10
	updatesPerGoroutine := 100

	sm := scanner.NewStateManager(200)

	// Concurrent updates
	start := time.Now()
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			for j := 0; j < updatesPerGoroutine; j++ {
				symbol := symbols[(id*updatesPerGoroutine+j)%symbolCount]
				tick := &models.Tick{
					Symbol:    symbol,
					Price:     100.0 + float64(j)*0.01,
					Size:      100,
					Timestamp: time.Now(),
					Type:      "trade",
				}
				if err := sm.UpdateLiveBar(symbol, tick); err != nil {
					t.Errorf("Failed to update live bar: %v", err)
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}

	elapsed := time.Since(start)
	totalUpdates := concurrency * updatesPerGoroutine
	rate := float64(totalUpdates) / elapsed.Seconds()

	t.Logf("Concurrent update test results:")
	t.Logf("  Total updates: %d", totalUpdates)
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Rate: %.2f updates/second", rate)

	// Verify all symbols are in state
	if sm.GetSymbolCount() != symbolCount {
		t.Errorf("Expected %d symbols in state, got %d", symbolCount, sm.GetSymbolCount())
	}
}

// BenchmarkScanLoop_2000Symbols benchmarks scan loop with 2000 symbols
func BenchmarkScanLoop_2000Symbols(b *testing.B) {
	symbolCount := 2000
	symbols := generateSymbols(symbolCount)
	sm := scanner.NewStateManager(200)

	// Initialize state
	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.Snapshot()
	}
}

// BenchmarkScanLoop_5000Symbols benchmarks scan loop with 5000 symbols
func BenchmarkScanLoop_5000Symbols(b *testing.B) {
	symbolCount := 5000
	symbols := generateSymbols(symbolCount)
	sm := scanner.NewStateManager(200)

	// Initialize state
	for _, symbol := range symbols {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     100.0,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		sm.UpdateLiveBar(symbol, tick)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sm.Snapshot()
	}
}

// generateSymbols generates a list of symbol names
func generateSymbols(count int) []string {
	symbols := make([]string, count)
	for i := 0; i < count; i++ {
		symbols[i] = fmt.Sprintf("SYMBOL%04d", i+1)
	}
	return symbols
}

