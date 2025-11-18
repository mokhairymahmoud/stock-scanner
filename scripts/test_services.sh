#!/bin/bash
# Comprehensive test script for all services

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================="
echo "Stock Scanner Services Test"
echo "==========================================${NC}"

# Check if services are running
check_service() {
    local service=$1
    local port=$2
    local name=$3
    
    echo -e "\n${YELLOW}Checking $name...${NC}"
    if curl -s -f "http://localhost:$port/health" > /dev/null; then
        echo -e "${GREEN}✓ $name is running${NC}"
        return 0
    else
        echo -e "${RED}✗ $name is not responding on port $port${NC}"
        return 1
    fi
}

# Test Ingest Service
test_ingest() {
    echo -e "\n${BLUE}=== Testing Ingest Service ===${NC}"
    
    if ! check_service "ingest" 8081 "Ingest Service"; then
        return 1
    fi
    
    # Check health endpoint
    echo -e "${YELLOW}Health check...${NC}"
    HEALTH=$(curl -s http://localhost:8081/health)
    echo "$HEALTH" | jq . || echo "$HEALTH"
    
    # Check provider connection
    PROVIDER_CONNECTED=$(echo "$HEALTH" | jq -r '.checks.provider.connected' 2>/dev/null || echo "false")
    if [ "$PROVIDER_CONNECTED" = "true" ]; then
        echo -e "${GREEN}✓ Provider is connected${NC}"
    else
        echo -e "${RED}✗ Provider is not connected${NC}"
        return 1
    fi
    
    # Check metrics
    echo -e "${YELLOW}Checking metrics...${NC}"
    METRICS=$(curl -s http://localhost:8081/metrics)
    PUBLISH_TOTAL=$(echo "$METRICS" | grep "stream_publish_total" | head -1 || echo "")
    if [ -n "$PUBLISH_TOTAL" ]; then
        echo -e "${GREEN}✓ Metrics available${NC}"
        echo "$PUBLISH_TOTAL"
    else
        echo -e "${YELLOW}⚠ No publish metrics yet (normal if no batches flushed)${NC}"
    fi
    
    # Check Redis stream
    echo -e "${YELLOW}Checking Redis stream...${NC}"
    if command -v redis-cli &> /dev/null; then
        STREAM_LENGTH=$(redis-cli XLEN ticks 2>/dev/null || echo "0")
        if [ "$STREAM_LENGTH" -gt 0 ]; then
            echo -e "${GREEN}✓ Stream has $STREAM_LENGTH messages${NC}"
        else
            echo -e "${YELLOW}⚠ No messages in stream yet (waiting for batch flush)${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ redis-cli not available, skipping stream check${NC}"
    fi
    
    return 0
}

# Test Bars Service
test_bars() {
    echo -e "\n${BLUE}=== Testing Bars Service ===${NC}"
    
    if ! check_service "bars" 8083 "Bars Service"; then
        return 1
    fi
    
    # Check health endpoint
    echo -e "${YELLOW}Health check...${NC}"
    HEALTH=$(curl -s http://localhost:8083/health)
    echo "$HEALTH" | jq . || echo "$HEALTH"
    
    # Check all components
    CONSUMER_RUNNING=$(echo "$HEALTH" | jq -r '.checks.consumer.running' 2>/dev/null || echo "false")
    PUBLISHER_RUNNING=$(echo "$HEALTH" | jq -r '.checks.publisher.running' 2>/dev/null || echo "false")
    DB_RUNNING=$(echo "$HEALTH" | jq -r '.checks.database.running' 2>/dev/null || echo "false")
    
    if [ "$CONSUMER_RUNNING" = "true" ]; then
        echo -e "${GREEN}✓ Consumer is running${NC}"
    else
        echo -e "${RED}✗ Consumer is not running${NC}"
    fi
    
    if [ "$PUBLISHER_RUNNING" = "true" ]; then
        echo -e "${GREEN}✓ Publisher is running${NC}"
    else
        echo -e "${RED}✗ Publisher is not running${NC}"
    fi
    
    if [ "$DB_RUNNING" = "true" ]; then
        echo -e "${GREEN}✓ Database client is running${NC}"
    else
        echo -e "${RED}✗ Database client is not running${NC}"
    fi
    
    # Check metrics
    echo -e "${YELLOW}Checking metrics...${NC}"
    METRICS=$(curl -s http://localhost:8083/metrics)
    CONSUME_TOTAL=$(echo "$METRICS" | grep "stream_consume_total" | head -1 || echo "")
    WRITE_TOTAL=$(echo "$METRICS" | grep "timescale_write_total" | head -1 || echo "")
    
    if [ -n "$CONSUME_TOTAL" ]; then
        echo -e "${GREEN}✓ Consumer metrics available${NC}"
        echo "$CONSUME_TOTAL"
    fi
    
    if [ -n "$WRITE_TOTAL" ]; then
        echo -e "${GREEN}✓ Database write metrics available${NC}"
        echo "$WRITE_TOTAL"
    fi
    
    # Check finalized bars stream
    echo -e "${YELLOW}Checking finalized bars stream...${NC}"
    if command -v redis-cli &> /dev/null; then
        STREAM_LENGTH=$(redis-cli XLEN bars.finalized 2>/dev/null || echo "0")
        if [ "$STREAM_LENGTH" -gt 0 ]; then
            echo -e "${GREEN}✓ Finalized bars stream has $STREAM_LENGTH messages${NC}"
        else
            echo -e "${YELLOW}⚠ No finalized bars yet (waiting for minute boundary)${NC}"
        fi
    fi
    
    return 0
}

# Test Data Flow
test_data_flow() {
    echo -e "\n${BLUE}=== Testing Data Flow ===${NC}"
    
    echo -e "${YELLOW}Waiting 30 seconds for data to flow...${NC}"
    sleep 30
    
    # Check ticks stream
    if command -v redis-cli &> /dev/null; then
        TICKS_COUNT=$(redis-cli XLEN ticks 2>/dev/null || echo "0")
        echo -e "${YELLOW}Ticks in stream: $TICKS_COUNT${NC}"
        
        if [ "$TICKS_COUNT" -gt 0 ]; then
            echo -e "${GREEN}✓ Ticks are being published${NC}"
        else
            echo -e "${RED}✗ No ticks in stream${NC}"
            return 1
        fi
    fi
    
    # Check finalized bars
    if command -v redis-cli &> /dev/null; then
        BARS_COUNT=$(redis-cli XLEN bars.finalized 2>/dev/null || echo "0")
        echo -e "${YELLOW}Finalized bars in stream: $BARS_COUNT${NC}"
        
        if [ "$BARS_COUNT" -gt 0 ]; then
            echo -e "${GREEN}✓ Bars are being finalized${NC}"
        else
            echo -e "${YELLOW}⚠ No finalized bars yet (may need to wait for minute boundary)${NC}"
        fi
    fi
    
    # Check database (if psql is available)
    if command -v psql &> /dev/null; then
        echo -e "${YELLOW}Checking database...${NC}"
        DB_COUNT=$(psql -h localhost -U postgres -d stock_scanner -t -c "SELECT COUNT(*) FROM bars_1m;" 2>/dev/null || echo "0")
        echo -e "${YELLOW}Bars in database: $DB_COUNT${NC}"
        
        if [ "$DB_COUNT" -gt 0 ]; then
            echo -e "${GREEN}✓ Bars are being written to database${NC}"
        else
            echo -e "${YELLOW}⚠ No bars in database yet (may need to wait longer)${NC}"
        fi
    fi
    
    return 0
}

# Main test execution
main() {
    INGEST_OK=false
    BARS_OK=false
    
    # Test Ingest Service
    if test_ingest; then
        INGEST_OK=true
    fi
    
    # Test Bars Service
    if test_bars; then
        BARS_OK=true
    fi
    
    # Test Data Flow
    test_data_flow
    
    # Summary
    echo -e "\n${BLUE}=========================================="
    echo "Test Summary"
    echo "==========================================${NC}"
    
    if [ "$INGEST_OK" = true ]; then
        echo -e "${GREEN}✓ Ingest Service: PASS${NC}"
    else
        echo -e "${RED}✗ Ingest Service: FAIL${NC}"
    fi
    
    if [ "$BARS_OK" = true ]; then
        echo -e "${GREEN}✓ Bars Service: PASS${NC}"
    else
        echo -e "${RED}✗ Bars Service: FAIL${NC}"
    fi
    
    if [ "$INGEST_OK" = true ] && [ "$BARS_OK" = true ]; then
        echo -e "\n${GREEN}All services are running correctly!${NC}"
        return 0
    else
        echo -e "\n${RED}Some services have issues. Check logs for details.${NC}"
        return 1
    fi
}

# Run tests
main

