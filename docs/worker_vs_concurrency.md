# Multiple Workers vs Single Worker with Go Concurrency

This document explains the key differences between using multiple workers (partitioning) versus using a single worker with Go concurrency.

## Current Implementation: Multiple Workers (Partitioning)

### Architecture

```
Worker 1 (goroutine/process)          Worker 2 (goroutine/process)
├── StateManager (symbols: A-F)       ├── StateManager (symbols: G-M)
├── ScanLoop                          ├── ScanLoop
├── Processes symbols sequentially    ├── Processes symbols sequentially
└── Own memory space                  └── Own memory space
```

### Key Characteristics

1. **Data Partitioning**: Each worker owns a subset of symbols via consistent hashing
   ```go
   // Worker 1 owns: AAPL, GOOGL, MSFT (hash % 2 == 0)
   // Worker 2 owns: TSLA, AMZN, NVDA (hash % 2 == 1)
   ```

2. **Separate State Managers**: Each worker has its own `StateManager` instance
   ```go
   worker1 := scanner.NewStateManager(200)  // Only has symbols A-F
   worker2 := scanner.NewStateManager(200)  // Only has symbols G-M
   ```

3. **Sequential Processing**: Each worker processes its symbols one by one
   ```go
   // In ScanLoop.Scan() - current implementation
   for _, symbol := range snapshot.Symbols {  // Sequential loop
       // Process symbol
   }
   ```

4. **Separate Memory Spaces**: Each worker's StateManager is independent
   - Less lock contention (each worker has its own locks)
   - Separate memory allocations
   - Can run on separate machines (distributed)

## Alternative: Single Worker with Go Concurrency

### Architecture

```
Single Worker
├── StateManager (ALL symbols: A-Z)
├── ScanLoop
└── Process symbols concurrently using goroutines
    ├── Goroutine 1: processes symbols A-F
    ├── Goroutine 2: processes symbols G-M
    ├── Goroutine 3: processes symbols N-S
    └── Goroutine 4: processes symbols T-Z
```

### Key Characteristics

1. **Shared State Manager**: All goroutines share the same `StateManager`
   ```go
   stateManager := scanner.NewStateManager(200)  // Has ALL symbols
   
   // All goroutines use the same instance
   go processSymbols(stateManager, symbolsA-F)
   go processSymbols(stateManager, symbolsG-M)
   ```

2. **Concurrent Processing**: Symbols processed in parallel using goroutines
   ```go
   // Hypothetical concurrent implementation
   var wg sync.WaitGroup
   for _, symbol := range snapshot.Symbols {
       wg.Add(1)
       go func(sym string) {
           defer wg.Done()
           // Process symbol concurrently
       }(symbol)
   }
   wg.Wait()
   ```

3. **Shared Memory Space**: All goroutines share the same heap
   - More lock contention (all goroutines compete for same locks)
   - Shared memory allocations
   - Limited to one machine

## Key Differences

### 1. Lock Contention

**Multiple Workers:**
```go
// Worker 1
stateManager1.mu.Lock()  // Only locks its own StateManager
// Process symbols A-F

// Worker 2  
stateManager2.mu.Lock()  // Different lock, no contention
// Process symbols G-M
```
- ✅ No lock contention between workers
- ✅ Each worker can proceed independently

**Single Worker with Concurrency:**
```go
// Goroutine 1
stateManager.mu.Lock()  // Locks shared StateManager
// Process symbols A-F

// Goroutine 2
stateManager.mu.Lock()  // WAITS for goroutine 1 to release
// Process symbols G-M
```
- ❌ High lock contention
- ❌ Goroutines block each other
- ❌ Defeats the purpose of concurrency

### 2. Memory and GC

**Multiple Workers:**
- Each worker has separate memory allocations
- GC can be more efficient (smaller heaps per worker)
- But: When workers run simultaneously, they all trigger GC together

**Single Worker with Concurrency:**
- All goroutines share the same heap
- Single GC affects all goroutines
- Larger heap = longer GC pauses

### 3. Scalability

**Multiple Workers:**
- ✅ Can scale horizontally (separate machines)
- ✅ Each machine has its own CPU, memory, GC
- ✅ True distributed scaling
- ✅ Fault isolation (one worker failure doesn't affect others)

**Single Worker with Concurrency:**
- ❌ Limited to one machine's resources
- ❌ Cannot scale beyond machine limits
- ❌ Single point of failure

### 4. Data Locality

**Multiple Workers:**
- Each worker only loads data for its symbols
- Better cache locality (smaller working set)
- Less memory per worker

**Single Worker with Concurrency:**
- Must load ALL symbols into memory
- Larger working set = more cache misses
- More memory required

### 5. Snapshot() Performance

**Multiple Workers:**
```go
// Worker 1: Snapshot() only copies symbols A-F (500k symbols)
snapshot1 := stateManager1.Snapshot()  // Smaller allocation

// Worker 2: Snapshot() only copies symbols G-M (500k symbols)  
snapshot2 := stateManager2.Snapshot()  // Smaller allocation
```
- Each snapshot is smaller
- But: When run simultaneously, they still compete for memory bandwidth

**Single Worker with Concurrency:**
```go
// Single snapshot copies ALL symbols (1M symbols)
snapshot := stateManager.Snapshot()  // Large allocation
// Then process concurrently
```
- One large snapshot
- All goroutines share the same snapshot (good!)
- But: Snapshot creation is a bottleneck

## Why Current Implementation Uses Multiple Workers

The current scanner uses multiple workers because:

1. **Horizontal Scalability**: Can run workers on separate machines
2. **Fault Isolation**: One worker failure doesn't affect others
3. **Resource Isolation**: Each worker has its own memory/GC
4. **Consistent Hashing**: Ensures each symbol is always processed by the same worker
5. **Production Reality**: In production, you'd run workers on separate machines/containers

## Could We Add Concurrency Within a Worker?

Yes! You could combine both approaches:

```go
// Multiple workers (horizontal scaling)
for workerID := 0; workerID < workerCount; workerID++ {
    go func(id int) {
        // Each worker processes its symbols concurrently
        var wg sync.WaitGroup
        for _, symbol := range workerSymbols[id] {
            wg.Add(1)
            go func(sym string) {
                defer wg.Done()
                // Process symbol
            }(symbol)
        }
        wg.Wait()
    }(workerID)
}
```

**Benefits:**
- ✅ Horizontal scaling (multiple workers)
- ✅ Vertical scaling (concurrency within each worker)
- ✅ Best of both worlds

**Challenges:**
- More complex
- Still need to handle lock contention within each worker
- Snapshot() would still be a bottleneck

## Summary

| Aspect | Multiple Workers | Single Worker + Concurrency |
|--------|------------------|----------------------------|
| **Scalability** | ✅ Horizontal (separate machines) | ❌ Vertical only (one machine) |
| **Lock Contention** | ✅ Low (separate locks) | ❌ High (shared locks) |
| **Memory** | ✅ Distributed | ❌ Single heap |
| **GC Impact** | ⚠️ Affects all workers on same machine | ❌ Affects all goroutines |
| **Fault Isolation** | ✅ Yes | ❌ No |
| **Data Locality** | ✅ Better (smaller working sets) | ❌ Worse (larger working set) |
| **Production Ready** | ✅ Yes (distributed) | ⚠️ Limited (single machine) |

## Conclusion

The current multiple-worker approach is better for production because:
1. It enables true horizontal scaling
2. Provides fault isolation
3. Reduces lock contention
4. Allows distributed deployment

Adding concurrency within each worker could provide additional benefits, but the main scaling strategy should remain multiple workers (partitioning).

