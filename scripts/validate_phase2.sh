#!/bin/bash
# Phase 2: Indicator Engine Validation Script

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${BLUE}=========================================="
echo "Phase 2: Indicator Engine Validation"
echo "==========================================${NC}"

# Check if Docker is running
if ! docker ps > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running${NC}"
    exit 1
fi

# Check if Redis container is running
if ! docker ps | grep -q stock-scanner-redis; then
    echo -e "${RED}❌ Redis container is not running${NC}"
    echo -e "${YELLOW}Start services with: make docker-up-all${NC}"
    exit 1
fi

# Function to check if service is responding
check_service() {
    local port=$1
    local name=$2
    
    if curl -s -f "http://localhost:$port/health" > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# 1. Check Indicator Service Health
echo -e "\n${CYAN}1. Checking Indicator Service Health...${NC}"
if ! check_service 8085 "Indicator Service"; then
    echo -e "${RED}❌ Indicator service is not responding on port 8085${NC}"
    echo -e "${YELLOW}Check if service is running: docker ps | grep indicator${NC}"
    exit 1
fi

HEALTH=$(curl -s http://localhost:8085/health)
STATUS=$(echo "$HEALTH" | jq -r '.status // "UNKNOWN"')
echo -e "${GREEN}✅ Service is responding${NC}"
echo -e "${YELLOW}Status: $STATUS${NC}"

if [ "$STATUS" != "UP" ]; then
    echo -e "${RED}❌ Service status is not UP${NC}"
    echo "$HEALTH" | jq .
    exit 1
fi

# Display health details
echo -e "\n${YELLOW}Health Details:${NC}"
echo "$HEALTH" | jq '{
    status: .status,
    consumer: .checks.consumer,
    engine: .checks.engine,
    publisher: .checks.publisher
}'

# 2. Check Finalized Bars Stream
echo -e "\n${CYAN}2. Checking Finalized Bars Stream...${NC}"
BARS_COUNT=$(docker exec stock-scanner-redis redis-cli XLEN bars.finalized 2>/dev/null || echo "0")
echo -e "${YELLOW}Finalized bars in stream: $BARS_COUNT${NC}"

if [ "$BARS_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✅ Bars are being finalized${NC}"
    
    # Show latest bar info
    echo -e "\n${YELLOW}Latest finalized bar:${NC}"
    docker exec stock-scanner-redis redis-cli XREVRANGE bars.finalized + - COUNT 1 2>/dev/null | head -5 || echo "Could not retrieve bar details"
else
    echo -e "${YELLOW}⚠️  No finalized bars yet (wait for minute boundary)${NC}"
    echo -e "${YELLOW}   This is normal if services just started. Wait 1-2 minutes.${NC}"
fi

# 3. Check Consumer Group
echo -e "\n${CYAN}3. Checking Consumer Group...${NC}"
CONSUMER_GROUP=$(docker exec stock-scanner-redis redis-cli XINFO GROUPS bars.finalized 2>/dev/null | grep -A 10 "indicator-engine" || echo "")
if [ -n "$CONSUMER_GROUP" ]; then
    echo -e "${GREEN}✅ Consumer group 'indicator-engine' exists${NC}"
    echo "$CONSUMER_GROUP" | head -10
else
    echo -e "${YELLOW}⚠️  Consumer group not found yet (may appear after first message)${NC}"
fi

# 4. Check Indicator Keys in Redis
echo -e "\n${CYAN}4. Checking Indicator Keys in Redis...${NC}"
IND_KEYS=$(docker exec stock-scanner-redis redis-cli KEYS "ind:*" 2>/dev/null | wc -l | tr -d ' ')
echo -e "${YELLOW}Indicator keys found: $IND_KEYS${NC}"

if [ "$IND_KEYS" -gt 0 ]; then
    echo -e "${GREEN}✅ Indicators are being published to Redis${NC}"
    
    # Show first indicator as example
    FIRST_KEY=$(docker exec stock-scanner-redis redis-cli KEYS "ind:*" 2>/dev/null | head -1 | tr -d '\r')
    if [ -n "$FIRST_KEY" ]; then
        echo -e "\n${YELLOW}Example indicator data ($FIRST_KEY):${NC}"
        IND_DATA=$(docker exec stock-scanner-redis redis-cli GET "$FIRST_KEY" 2>/dev/null)
        if command -v jq &> /dev/null; then
            echo "$IND_DATA" | jq . 2>/dev/null || echo "$IND_DATA"
        else
            echo "$IND_DATA"
        fi
        
        # Count indicators in the values
        if command -v jq &> /dev/null; then
            IND_COUNT=$(echo "$IND_DATA" | jq '.values | length' 2>/dev/null || echo "0")
            echo -e "${YELLOW}Number of indicators computed: $IND_COUNT${NC}"
        fi
    fi
    
    # List all symbols with indicators
    echo -e "\n${YELLOW}Symbols with indicators:${NC}"
    docker exec stock-scanner-redis redis-cli KEYS "ind:*" 2>/dev/null | sed 's/ind://' | sort
else
    echo -e "${YELLOW}⚠️  No indicators yet${NC}"
    echo -e "${YELLOW}   This is normal if:${NC}"
    echo -e "${YELLOW}   - Services just started (wait 1-2 minutes)${NC}"
    echo -e "${YELLOW}   - No bars have been finalized yet${NC}"
    echo -e "${YELLOW}   - Indicator calculators need more data (RSI needs 15 bars)${NC}"
fi

# 5. Check Symbol Count
echo -e "\n${CYAN}5. Checking Engine Symbol Count...${NC}"
SYMBOL_COUNT=$(echo "$HEALTH" | jq -r '.checks.engine.symbol_count // 0')
echo -e "${YELLOW}Symbols being tracked: $SYMBOL_COUNT${NC}"

if [ "$SYMBOL_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✅ Engine is processing $SYMBOL_COUNT symbol(s)${NC}"
else
    echo -e "${YELLOW}⚠️  No symbols tracked yet (wait for bars to be processed)${NC}"
fi

# 6. Check Consumer Stats
echo -e "\n${CYAN}6. Checking Consumer Statistics...${NC}"
CONSUMER_STATS=$(echo "$HEALTH" | jq '.checks.consumer.stats // {}')
if [ "$(echo "$CONSUMER_STATS" | jq 'keys | length')" -gt 0 ]; then
    echo -e "${GREEN}✅ Consumer statistics available${NC}"
    echo "$CONSUMER_STATS" | jq .
else
    echo -e "${YELLOW}⚠️  Consumer stats not available yet${NC}"
fi

# 7. Check Metrics
echo -e "\n${CYAN}7. Checking Prometheus Metrics...${NC}"
METRICS=$(curl -s http://localhost:8085/metrics 2>/dev/null || echo "")
if [ -n "$METRICS" ]; then
    IND_METRICS=$(echo "$METRICS" | grep -i "indicator" | head -5 || echo "")
    if [ -n "$IND_METRICS" ]; then
        echo -e "${GREEN}✅ Indicator metrics are available${NC}"
        echo "$IND_METRICS"
    else
        echo -e "${YELLOW}⚠️  No indicator-specific metrics found${NC}"
        echo -e "${YELLOW}   (Metrics may use different naming)${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Could not retrieve metrics${NC}"
fi

# 8. Check Pub/Sub Channel (optional - just info)
echo -e "\n${CYAN}8. Pub/Sub Channel Info...${NC}"
echo -e "${YELLOW}To monitor indicator updates in real-time, run:${NC}"
echo -e "${CYAN}docker exec -it stock-scanner-redis redis-cli PSUBSCRIBE 'indicators.updated'${NC}"

# 9. Check Dependencies
echo -e "\n${CYAN}9. Checking Service Dependencies...${NC}"

# Check Bars service
if check_service 8083 "Bars Service"; then
    echo -e "${GREEN}✅ Bars service is running${NC}"
    BARS_HEALTH=$(curl -s http://localhost:8083/health | jq -r '.status // "UNKNOWN"')
    echo -e "${YELLOW}   Bars service status: $BARS_HEALTH${NC}"
else
    echo -e "${RED}❌ Bars service is not running${NC}"
    echo -e "${YELLOW}   Indicator service depends on bars service${NC}"
fi

# Check Ingest service
if check_service 8081 "Ingest Service"; then
    echo -e "${GREEN}✅ Ingest service is running${NC}"
else
    echo -e "${YELLOW}⚠️  Ingest service is not running (indicator service can still work with existing bars)${NC}"
fi

# Summary
echo -e "\n${BLUE}=========================================="
echo "Validation Summary"
echo "==========================================${NC}"

ALL_OK=true

if [ "$STATUS" != "UP" ]; then
    echo -e "${RED}❌ Indicator service is not UP${NC}"
    ALL_OK=false
else
    echo -e "${GREEN}✅ Indicator service is UP${NC}"
fi

if [ "$IND_KEYS" -gt 0 ]; then
    echo -e "${GREEN}✅ Indicators are being published ($IND_KEYS keys)${NC}"
else
    echo -e "${YELLOW}⚠️  No indicators published yet${NC}"
    if [ "$BARS_COUNT" -eq 0 ]; then
        echo -e "${YELLOW}   Reason: No finalized bars available${NC}"
    else
        echo -e "${YELLOW}   Reason: May need more bars for indicators to be ready${NC}"
    fi
    ALL_OK=false
fi

if [ "$SYMBOL_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✅ Engine is tracking $SYMBOL_COUNT symbol(s)${NC}"
else
    echo -e "${YELLOW}⚠️  Engine not tracking any symbols yet${NC}"
    ALL_OK=false
fi

echo -e "\n${CYAN}Next Steps:${NC}"
if [ "$ALL_OK" = true ]; then
    echo -e "${GREEN}✅ Phase 2 validation PASSED!${NC}"
    echo -e "${CYAN}   - Indicators are being computed and published${NC}"
    echo -e "${CYAN}   - Check RedisInsight at http://localhost:8001 for visual inspection${NC}"
    echo -e "${CYAN}   - Monitor logs: docker logs -f stock-scanner-indicator${NC}"
else
    echo -e "${YELLOW}⚠️  Some checks need attention${NC}"
    echo -e "${CYAN}   - Wait 1-2 minutes for data to flow${NC}"
    echo -e "${CYAN}   - Check service logs: docker logs stock-scanner-indicator${NC}"
    echo -e "${CYAN}   - Verify bars service is running: curl http://localhost:8083/health${NC}"
    echo -e "${CYAN}   - Check finalized bars: docker exec stock-scanner-redis redis-cli XLEN bars.finalized${NC}"
fi

echo ""

