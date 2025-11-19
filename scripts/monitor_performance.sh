#!/bin/bash
# Performance monitoring script for ingest service

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=========================================="
echo "Ingest Service Performance Monitor"
echo "==========================================${NC}"

# Get metrics
METRICS=$(curl -s http://localhost:8081/metrics 2>/dev/null || echo "")

if [ -z "$METRICS" ]; then
    echo -e "${RED}Error: Cannot connect to ingest service at http://localhost:8081/metrics${NC}"
    echo "Make sure the service is running: docker-compose -f config/docker-compose.yaml ps ingest"
    exit 1
fi

# Extract metrics
PUBLISH_TOTAL=$(echo "$METRICS" | grep "^stream_publish_total" | head -1 | awk '{print $2}' || echo "0")
PUBLISH_ERRORS=$(echo "$METRICS" | grep "^stream_publish_errors_total" | head -1 | awk '{print $2}' || echo "0")
BATCH_SIZE=$(echo "$METRICS" | grep "^stream_publish_batch_size" | head -1 | awk '{print $2}' || echo "0")
LATENCY=$(echo "$METRICS" | grep "^stream_publish_latency_seconds" | grep -v bucket | grep -v count | grep -v sum | head -1 | awk '{print $2}' || echo "0")

# Get Redis stream length
STREAM_LENGTH=$(docker exec stock-scanner-redis redis-cli XLEN ticks 2>/dev/null | tail -1 || echo "0")

# Get provider status
PROVIDER_STATUS=$(curl -s http://localhost:8081/health 2>/dev/null | jq -r '.checks.provider.connected' 2>/dev/null || echo "unknown")
PROVIDER_NAME=$(curl -s http://localhost:8081/health 2>/dev/null | jq -r '.checks.provider.provider' 2>/dev/null || echo "unknown")

# Get subscribed symbols count
SYMBOLS=$(curl -s http://localhost:8081/health 2>/dev/null | jq -r '.checks.provider.symbols // [] | length' 2>/dev/null || echo "0")

# Calculate expected rate (10 ticks/sec per symbol for mock provider)
if [ "$PROVIDER_NAME" = "mock" ]; then
    EXPECTED_RATE=$((SYMBOLS * 10))
else
    EXPECTED_RATE="N/A"
fi

echo -e "\n${YELLOW}Provider Status:${NC}"
echo -e "  Provider: ${GREEN}$PROVIDER_NAME${NC}"
echo -e "  Connected: ${GREEN}$PROVIDER_STATUS${NC}"
echo -e "  Symbols: ${GREEN}$SYMBOLS${NC}"
if [ "$PROVIDER_NAME" = "mock" ]; then
    echo -e "  Expected Rate: ${GREEN}$EXPECTED_RATE ticks/sec${NC}"
fi

echo -e "\n${YELLOW}Publish Metrics:${NC}"
echo -e "  Total Published: ${GREEN}$PUBLISH_TOTAL${NC}"
echo -e "  Errors: ${RED}$PUBLISH_ERRORS${NC}"
echo -e "  Avg Batch Size: ${GREEN}$BATCH_SIZE${NC}"
if [ "$LATENCY" != "0" ] && [ -n "$LATENCY" ]; then
    LATENCY_MS=$(echo "$LATENCY * 1000" | bc 2>/dev/null || echo "N/A")
    echo -e "  Avg Latency: ${GREEN}${LATENCY_MS}ms${NC}"
fi

echo -e "\n${YELLOW}Redis Stream:${NC}"
echo -e "  Stream Length: ${GREEN}$STREAM_LENGTH${NC}"

# Calculate rate if we have previous values
if [ -f /tmp/ingest_perf_prev ]; then
    PREV_TOTAL=$(cat /tmp/ingest_perf_prev | cut -d' ' -f1)
    PREV_TIME=$(cat /tmp/ingest_perf_prev | cut -d' ' -f2)
    CURRENT_TIME=$(date +%s)
    
    if [ "$PREV_TOTAL" -gt 0 ] && [ "$CURRENT_TIME" -gt "$PREV_TIME" ]; then
        DELTA=$((PUBLISH_TOTAL - PREV_TOTAL))
        DELTA_TIME=$((CURRENT_TIME - PREV_TIME))
        if [ "$DELTA_TIME" -gt 0 ]; then
            RATE=$((DELTA / DELTA_TIME))
            echo -e "\n${YELLOW}Current Rate:${NC}"
            echo -e "  ${GREEN}$RATE ticks/sec${NC}"
            
            if [ "$PROVIDER_NAME" = "mock" ] && [ "$EXPECTED_RATE" != "N/A" ]; then
                if [ "$RATE" -ge "$EXPECTED_RATE" ]; then
                    echo -e "  ${GREEN}✓ Meeting expected rate${NC}"
                else
                    PERCENT=$((RATE * 100 / EXPECTED_RATE))
                    echo -e "  ${YELLOW}⚠ Only $PERCENT% of expected rate${NC}"
                fi
            fi
        fi
    fi
fi

# Save current values
echo "$PUBLISH_TOTAL $(date +%s)" > /tmp/ingest_perf_prev

echo -e "\n${BLUE}=========================================="
echo "Run this script again in a few seconds to see rate"
echo "==========================================${NC}"

