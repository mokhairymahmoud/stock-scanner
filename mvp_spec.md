# Real-Time Trading Scanner MVP — Technical Specification

## Overview
This document describes the architecture, services, data flows, and design constraints for a real-time stock market scanning system similar to DayTradingDash or Trade Ideas. It is intended for GitHub Copilot and other AI coding assistants to generate project code.

## Goals
- Sub‑second scanning of 2,000–10,000 symbols.
- Real-time tick ingestion.
- Rule-based scanner engine.
- Alert publishing (WebSocket + Kafka).
- Minimal latency, horizontally scalable workers.

## Architecture Summary
### Services
1. Market Data Ingest  
2. Bar Aggregator (1m bars)  
3. Indicator Engine (async computations)  
4. Scanner Worker (parallel microservices)  
5. Rule Engine (manages rules)  
6. WebSocket Gateway  
7. API Service (REST)  
8. Storage:
   - Redis (real-time state)
   - TimescaleDB (bars)
   - ClickHouse (historical analytics)
   - S3 (cold storage)

## Data Flow
1. Ingest receives ticks → publishes to Redis/Kafka.
2. Bar Aggregator updates live 1m bar in Redis, writes completed bars to TimescaleDB.
3. Indicator Engine consumes bars → publishes indicators to Redis.
4. Scanner Workers subscribe to tick & indicator updates:
   - All data kept fully in-memory.
   - Each worker assigned a partition of symbols.
5. Workers evaluate rules every second.
6. Matches published to:
   - Redis pubsub
   - WebSocket Gateway
   - Alerts queue (email/SMS)

## Key Constraints
- No DB calls in scan loop.
- No Redis GET inside scan loop.
- All data kept in memory: ticks, bars, indicators, rules.
- Workers scaled horizontally via symbol partitioning.
- Hard limit: scanning cycle must complete < 800ms.

## Directory Structure (Recommended)
```
/cmd
  /ingest
  /scanner
  /bars
  /ws_gateway
  /api

/internal
  /config
  /data
  /rules
  /scanner
  /models
  /pubsub
  /storage

/pkg
  /indicator
  /timeseries
  /logger

/scripts
  /migrations

/docs
  architecture.md
  api.md

/config
  docker-compose.yaml
  env.example
```

## Scanner Worker Logic
1. Load assigned symbols.
2. Sub to tick + indicator streams.
3. Maintain in-memory map:
```
symbolData[symbol] = {
    lastPrice,
    volume,
    bar1m,
    indicators: { rsi, vwap, ma20, ... }
}
```
4. On each scan interval (1s):
```
for symbol in partition:
    d = symbolData[symbol]
    for rule in rules:
        if rule applies:
            publish match
```

## Rule Format (JSON)
```
{
  "id": "rv-gt-2",
  "name": "Relative Volume > 2",
  "type": "numeric",
  "metric": "relative_volume",
  "operator": ">",
  "value": 2.0
}
```

## Technologies
- Go 1.22+
- Redis (pubsub, in-memory state)
- Kafka (optional)
- TimescaleDB
- ClickHouse
- WebSockets
- Docker / Kubernetes

## Next Steps
- Implement ingest → Redis stream.
- Implement bar aggregator.
- Implement rule engine.
- Implement scanner workers with <1s cycle.
- Implement WebSocket gateway.

