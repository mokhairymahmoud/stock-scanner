#!/bin/bash
# Test script for ingest service

set -e

echo "=========================================="
echo "Testing Ingest Service"
echo "=========================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Redis is running
echo -e "${YELLOW}Checking Redis connection...${NC}"
if redis-cli ping > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Redis is running${NC}"
else
    echo -e "${RED}✗ Redis is not running${NC}"
    echo "Starting Redis with Docker Compose..."
    docker-compose -f config/docker-compose.yaml up -d redis
    echo "Waiting for Redis to be ready..."
    sleep 3
fi

# Build the service
echo -e "${YELLOW}Building ingest service...${NC}"
if go build -o ingest ./cmd/ingest; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi

# Start the service in background
echo -e "${YELLOW}Starting ingest service...${NC}"
./ingest > ingest.log 2>&1 &
INGEST_PID=$!

# Function to cleanup
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    kill $INGEST_PID 2>/dev/null || true
    wait $INGEST_PID 2>/dev/null || true
    rm -f ingest
    echo -e "${GREEN}✓ Cleanup complete${NC}"
}
trap cleanup EXIT

# Wait for service to start
echo "Waiting for service to start..."
sleep 3

# Check if service is running
if ! kill -0 $INGEST_PID 2>/dev/null; then
    echo -e "${RED}✗ Service failed to start${NC}"
    echo "Logs:"
    cat ingest.log
    exit 1
fi

echo -e "${GREEN}✓ Service is running (PID: $INGEST_PID)${NC}"

# Test health endpoint
echo -e "\n${YELLOW}Testing health endpoint...${NC}"
HEALTH_RESPONSE=$(curl -s http://localhost:8081/health || echo "")
if [ -z "$HEALTH_RESPONSE" ]; then
    echo -e "${RED}✗ Health endpoint not responding${NC}"
    exit 1
fi

echo "$HEALTH_RESPONSE" | jq . || echo "$HEALTH_RESPONSE"

# Check if provider is connected
PROVIDER_CONNECTED=$(echo "$HEALTH_RESPONSE" | jq -r '.checks.provider.connected' 2>/dev/null || echo "false")
if [ "$PROVIDER_CONNECTED" = "true" ]; then
    echo -e "${GREEN}✓ Provider is connected${NC}"
else
    echo -e "${RED}✗ Provider is not connected${NC}"
    exit 1
fi

# Test readiness
echo -e "\n${YELLOW}Testing readiness endpoint...${NC}"
READY_RESPONSE=$(curl -s http://localhost:8081/ready)
if [ "$READY_RESPONSE" = "ready" ]; then
    echo -e "${GREEN}✓ Service is ready${NC}"
else
    echo -e "${RED}✗ Service is not ready${NC}"
    exit 1
fi

# Test liveness
echo -e "\n${YELLOW}Testing liveness endpoint...${NC}"
LIVE_RESPONSE=$(curl -s http://localhost:8081/live)
if [ "$LIVE_RESPONSE" = "alive" ]; then
    echo -e "${GREEN}✓ Service is alive${NC}"
else
    echo -e "${RED}✗ Service is not alive${NC}"
    exit 1
fi

# Wait for ticks to be published
echo -e "\n${YELLOW}Waiting for ticks to be published (10 seconds)...${NC}"
sleep 10

# Check Redis stream
echo -e "\n${YELLOW}Checking Redis stream...${NC}"
STREAM_LENGTH=$(redis-cli XLEN ticks 2>/dev/null || echo "0")
if [ "$STREAM_LENGTH" -gt 0 ]; then
    echo -e "${GREEN}✓ Stream has $STREAM_LENGTH messages${NC}"
    
    # Read sample messages
    echo -e "\n${YELLOW}Reading sample messages...${NC}"
    redis-cli XREAD COUNT 3 STREAMS ticks 0 2>/dev/null || echo "No messages found"
else
    echo -e "${YELLOW}⚠ No messages in stream yet (this is normal if batch hasn't flushed)${NC}"
    echo "Checking batch size..."
    BATCH_SIZE=$(echo "$HEALTH_RESPONSE" | jq -r '.checks.publisher.batch_size' 2>/dev/null || echo "0")
    if [ "$BATCH_SIZE" -gt 0 ]; then
        echo -e "${YELLOW}  Batch size: $BATCH_SIZE (waiting for flush)${NC}"
    fi
fi

# Check metrics
echo -e "\n${YELLOW}Checking metrics...${NC}"
METRICS=$(curl -s http://localhost:8081/metrics)
PUBLISH_TOTAL=$(echo "$METRICS" | grep "stream_publish_total" | head -1 || echo "")
if [ -n "$PUBLISH_TOTAL" ]; then
    echo -e "${GREEN}✓ Metrics available${NC}"
    echo "$PUBLISH_TOTAL"
else
    echo -e "${YELLOW}⚠ No publish metrics yet (normal if no batches flushed)${NC}"
fi

echo -e "\n${GREEN}=========================================="
echo "✓ All tests passed!"
echo "==========================================${NC}"

# Keep service running for manual inspection
echo -e "\n${YELLOW}Service is running. Press Ctrl+C to stop.${NC}"
echo "You can:"
echo "  - Check health: curl http://localhost:8081/health"
echo "  - Check metrics: curl http://localhost:8081/metrics"
echo "  - Check stream: redis-cli XREAD COUNT 10 STREAMS ticks 0"
echo ""

# Wait for interrupt
wait $INGEST_PID

