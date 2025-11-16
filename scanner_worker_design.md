# Scanner Worker Design — Deep Technical Spec

This document details the internal design of a single Scanner Worker process: data structures, ingestion model, rule compilation, scan loop, cooldown/dedupe, metrics, testing, and deployment guidance.

---

## Goals
- Evaluate rules across assigned symbols every 1 second.
- Keep hot-path operations strictly memory-bound (no network DB calls).
- Support thousands of symbols per worker depending on CPU.
- Provide deterministic latency and observability.

---

## Ingestion Model

### Channels
- `tickCh` (buffered channel): receives Tick messages from dispatcher (in-process) or sub from Redis/Kafka assigned partitions.
- `barFinalizedCh` (buffered): receives finalized 1m bars (for indicator window updates).
- `controlCh`: admin commands (reload rules, change universe).

### Ownership
- Worker subscribes to a set of partitions via Kafka consumer group or to Redis Stream group.
- Ownership must guarantee symbol-to-worker affinity.

---

## In-Memory Data Structures

```go
type LiveBar struct {
    Time time.Time
    Symbol string
    Open float64
    High float64
    Low float64
    Close float64
    Volume int64
    vwapNum float64
    vwapDen float64
}

type SymbolState struct {
    live *LiveBar
    lastFinalBars []Bar1m  // fixed-size ring buffer of last N finalized 1m bars
    indicators map[string]float64 // precomputed (rsi, ema20, vwap, avg5mVol...)
    lastTick time.Time
    mu sync.RWMutex
}
```

- `live` updated on every tick.
- `lastFinalBars` updated when a finalized bar event arrives.
- `indicators` updated by indicator engine or computed incrementally.

---

## Rule Representation & Compilation

### Rule JSON (example)
```json
{
  "id":"r1",
  "name":"Momentum5m",
  "conditions":[
    {"metric":"price_change_5m_pct","op":">","value":0.5},
    {"metric":"rel_volume_5m","op":">","value":1.5}
  ],
  "cooldown_sec":20
}
```

### Compilation
- Parse JSON to AST.
- Validate metrics referenced exist or can be computed from state.
- Compile into a small bytecode or closure in Go to avoid repeated parsing.
  - e.g. `type CompiledRule func(*SymbolState) bool`

### Example compile step (pseudo-Go)
```go
func compileRule(r Rule) (CompiledRule, error) {
   // create closure capturing thresholds and ops
   return func(s *SymbolState) bool {
       // compute necessary metrics using s.indicators and s.live
       return cond1 && cond2 && ...
   }, nil
}
```

This makes evaluation a pure in-memory function call per symbol.

---

## Scan Loop (hot path)

Pseudocode:
```
ticker := time.NewTicker(1s)
for {
  <-ticker.C
  snapshot := snapshotSymbolsList()
  for _, sym := range snapshot {
    state := states[sym]
    for _, cr := range compiledRules {
      if cr(state) {
         if not onCooldown(rule, sym) {
            emitAlert(rule, sym, state)
         }
      }
    }
  }
}
```

Key points:
- `snapshotSymbolsList()` copies keys under RLock quickly.
- Evaluation uses RLock per symbol only briefly for reading metrics.
- Avoid allocations inside loop: reuse buffers and primitives.

---

## Cooldown & Deduplication

- Use `lastFired map[string]time.Time` keyed by `ruleID|symbol`.
- When a rule matches, check `now - lastFired < cooldown` → skip.
- Store alert UUIDs and idempotency keys in ClickHouse or Redis for cross-worker dedupe (if multiple workers could generate duplicates during resharding).

---

## Alert Emission

- Alerts are small JSON objects published to `alerts` Redis channel or Kafka topic.
- Include `alert_id`, `rule_id`, `symbol`, `price`, `time`, `trace_id`.
- Alert Service will persist and deliver.

---

## Observability & Metrics

Expose Prometheus metrics:
- `scan_cycle_seconds` (histogram)
- `symbols_scanned_total`
- `alerts_emitted_total`
- `tick_processed_total`
- `worker_queue_depth`
- `last_tick_timestamp{symbol}` (optional gauge for hot symbols)

Traces:
- Trace when alert is emitted to measure end-to-end latency (tick -> alert -> websocket).

---

## Failure Modes & Recovery

- **Worker crash**: consumer group rebalances; new worker rehydrates state from Redis and Timescale.
- **Stock spikes**: inbound tick rate increases; use larger tickCh buffer and autoscale workers; shed low-priority symbols if necessary.
- **State corruption**: restart worker; rehydrate from `livebar:{symbol}` key in Redis and last finalized bars from Timescale.

---

## Testing Strategy

- Unit tests: compileRule correctness, metrics computations, compare ops.
- Integration tests: mock ingest + worker; assert alerts emitted for crafted scenarios.
- Load tests: simulate 10k symbols with tick bursts; measure scan_cycle_seconds distribution.
- Chaos tests: simulate Kafka partition reassignment and verify no duplicate alerts (idempotency).

---

## Deployment Guidance

- Dockerize worker binary.
- Use Kubernetes Deployment with HPA based on CPU and `worker_queue_depth` metric.
- Start with conservative `symbols_per_worker` to tune CPU.
- Use readinessProbe that ensures worker has rehydrated minimal state before receiving live traffic.

---

## Code Patterns & Optimizations

- Avoid per-symbol heap allocations in hot loop; reuse objects via sync.Pool.
- Use structs and arrays instead of maps when symbol set is fixed/known.
- Minimize locking: use RLock for readers; perform writes only in tick handler.
- Consider a native shared memory segment or memory-mapped file for extremely low-latency cross-process state (advanced).

---

## Example: Compiled Rule Closure (Go)
```go
func compileMomentumRule(threshold float64) CompiledRule {
  return func(s *SymbolState) bool {
     p := s.live.Close
     p5 := s.lastFinalBars[len(s.lastFinalBars)-5].Close // guard indices in real code
     pct := (p - p5) / p5 * 100
     relVol := float64(s.live.Volume) / s.indicators["avg5mVol"]
     return pct > threshold && relVol > 1.5
  }
}
```

---

## Next Steps for Engineers

1. Implement tick ingestion and dispatcher (single-node).
2. Implement worker skeleton with tick handler, state map, compiled rule evaluation.
3. Implement bar aggregator (finalize & persist to Timescale).
4. Add Prometheus metrics and basic alert service.
5. Run local load tests and tune worker buffer sizes and number of workers.

