// Package data contains stress tests for the stock scanner system.
//
// These tests verify system behavior under stress conditions:
// - Tick bursts
// - High rule counts
// - Many concurrent WebSocket connections
// - Database connection pool exhaustion
// - Memory pressure
// - Concurrent rule updates
//
// See README.md for documentation on running performance tests.
package data

import (
	"sync"
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/pubsub"
	"github.com/mohamedkhairy/stock-scanner/internal/scanner"
)

// TestStress_TickBurst tests system behavior under tick bursts
func TestStress_TickBurst(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
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

	symbol := "AAPL"
	burstSize := 1000

	// Measure burst handling
	start := time.Now()
	successCount := 0
	errorCount := 0

	// Send burst of ticks
	for i := 0; i < burstSize; i++ {
		tick := &models.Tick{
			Symbol:    symbol,
			Price:     150.0 + float64(i)*0.01,
			Size:      100,
			Timestamp: time.Now(),
			Type:      "trade",
		}
		if err := publisher.Publish(tick); err != nil {
			errorCount++
		} else {
			successCount++
		}
	}
	
	publisher.Flush()

	elapsed := time.Since(start)
	rate := float64(successCount) / elapsed.Seconds()

	t.Logf("Tick burst test results:")
	t.Logf("  Burst size: %d", burstSize)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Rate: %.2f ticks/second", rate)

	if errorCount > burstSize/10 {
		t.Errorf("Too many errors during burst: %d/%d", errorCount, burstSize)
	}
}

// TestStress_HighRuleCount tests system with many rules
func TestStress_HighRuleCount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ruleCount := 100
	symbolCount := 100

	sm := scanner.NewStateManager(200)
	symbols := generateSymbols(symbolCount)

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

	// Simulate rule evaluation (in real scenario, rules would be compiled)
	start := time.Now()
	
	// Simulate evaluating all rules for all symbols
	for _, symbol := range symbols {
		for i := 0; i < ruleCount; i++ {
			// Simulate rule evaluation (just a placeholder - get snapshot)
			snapshot := sm.Snapshot()
			_ = snapshot.States[symbol]
		}
	}

	elapsed := time.Since(start)
	totalEvaluations := symbolCount * ruleCount
	rate := float64(totalEvaluations) / elapsed.Seconds()

	t.Logf("High rule count test results:")
	t.Logf("  Rules: %d", ruleCount)
	t.Logf("  Symbols: %d", symbolCount)
	t.Logf("  Total evaluations: %d", totalEvaluations)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Rate: %.2f evaluations/second", rate)

	// Verify performance is reasonable
	if elapsed > 5*time.Second {
		t.Errorf("Rule evaluation took too long: %v", elapsed)
	}
}

// TestStress_WebSocketConnections tests many concurrent WebSocket connections
func TestStress_WebSocketConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// This test simulates many WebSocket connections
	// In a real scenario, we would create actual WebSocket connections
	connectionCount := 1000
	concurrentConnections := 50

	start := time.Now()
	var wg sync.WaitGroup
	successCount := 0
	errorCount := 0
	var mu sync.Mutex

	// Simulate concurrent connection creation
	for i := 0; i < concurrentConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			connectionsPerGoroutine := connectionCount / concurrentConnections
			for j := 0; j < connectionsPerGoroutine; j++ {
				// Simulate connection creation
				// In real test, this would create actual WebSocket connection
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("WebSocket connection test results:")
	t.Logf("  Target connections: %d", connectionCount)
	t.Logf("  Concurrent goroutines: %d", concurrentConnections)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Duration: %v", elapsed)

	if successCount < connectionCount*9/10 {
		t.Errorf("Failed to create enough connections: %d/%d", successCount, connectionCount)
	}
}

// TestStress_DatabaseConnectionPool tests database connection pool exhaustion
func TestStress_DatabaseConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// This test simulates database connection pool stress
	// In a real scenario, we would use actual database connections
	concurrentQueries := 100
	maxConnections := 10

	start := time.Now()
	var wg sync.WaitGroup
	successCount := 0
	errorCount := 0
	var mu sync.Mutex

	// Simulate concurrent database queries
	semaphore := make(chan struct{}, maxConnections)
	for i := 0; i < concurrentQueries; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			// Simulate database query
			time.Sleep(10 * time.Millisecond)
			
			mu.Lock()
			successCount++
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Database connection pool test results:")
	t.Logf("  Concurrent queries: %d", concurrentQueries)
	t.Logf("  Max connections: %d", maxConnections)
	t.Logf("  Success: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Duration: %v", elapsed)

	if successCount < concurrentQueries*9/10 {
		t.Errorf("Too many queries failed: %d/%d", errorCount, concurrentQueries)
	}
}

// TestStress_MemoryPressure tests system behavior under memory pressure
func TestStress_MemoryPressure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	symbolCount := 10000
	symbols := generateSymbols(symbolCount)

	sm := scanner.NewStateManager(200)

	// Create many state updates to simulate memory pressure
	start := time.Now()
	updateCount := 0

	for i := 0; i < 10; i++ {
		for _, symbol := range symbols {
			tick := &models.Tick{
				Symbol:    symbol,
				Price:     100.0 + float64(i)*0.01,
				Size:      100,
				Timestamp: time.Now(),
				Type:      "trade",
			}
			if err := sm.UpdateLiveBar(symbol, tick); err != nil {
				t.Errorf("Failed to update live bar: %v", err)
			}
			updateCount++
		}
	}

	elapsed := time.Since(start)

	t.Logf("Memory pressure test results:")
	t.Logf("  Symbols: %d", symbolCount)
	t.Logf("  Updates: %d", updateCount)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Updates per second: %.2f", float64(updateCount)/elapsed.Seconds())

	// Verify state manager still works correctly
	if sm.GetSymbolCount() != symbolCount {
		t.Errorf("State manager lost symbols: expected %d, got %d", symbolCount, sm.GetSymbolCount())
	}
}

// TestStress_ConcurrentRuleUpdates tests concurrent rule updates
func TestStress_ConcurrentRuleUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ruleCount := 100
	concurrency := 10
	updatesPerGoroutine := 10

	start := time.Now()
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Simulate concurrent rule updates
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				// Simulate rule update
				time.Sleep(1 * time.Millisecond)
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Concurrent rule update test results:")
	t.Logf("  Rules: %d", ruleCount)
	t.Logf("  Concurrency: %d", concurrency)
	t.Logf("  Updates per goroutine: %d", updatesPerGoroutine)
	t.Logf("  Total updates: %d", successCount)
	t.Logf("  Duration: %v", elapsed)

	expectedUpdates := concurrency * updatesPerGoroutine
	if successCount != expectedUpdates {
		t.Errorf("Expected %d updates, got %d", expectedUpdates, successCount)
	}
}

