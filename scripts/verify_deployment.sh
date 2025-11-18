#!/bin/bash
# Quick verification script for deployed services

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=========================================="
echo "Service Deployment Verification"
echo "==========================================${NC}"

# Check if services are running
check_health() {
    local service=$1
    local port=$2
    local name=$3
    
    if curl -s -f "http://localhost:$port/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ $name (port $port)${NC}"
        return 0
    else
        echo -e "${RED}✗ $name (port $port) - NOT RESPONDING${NC}"
        return 1
    fi
}

# Check infrastructure
check_infrastructure() {
    echo -e "\n${YELLOW}Infrastructure Services:${NC}"
    
    # Redis
    if docker exec stock-scanner-redis redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Redis${NC}"
    else
        echo -e "${RED}✗ Redis${NC}"
    fi
    
    # TimescaleDB
    if docker exec stock-scanner-timescaledb pg_isready -U postgres > /dev/null 2>&1; then
        echo -e "${GREEN}✓ TimescaleDB${NC}"
    else
        echo -e "${RED}✗ TimescaleDB${NC}"
    fi
    
    # Prometheus
    if curl -s -f "http://localhost:9090/-/healthy" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Prometheus${NC}"
    else
        echo -e "${RED}✗ Prometheus${NC}"
    fi
}

# Check Go services
check_go_services() {
    echo -e "\n${YELLOW}Go Services:${NC}"
    
    INGEST_OK=false
    BARS_OK=false
    
    if check_health "ingest" 8081 "Ingest Service"; then
        INGEST_OK=true
    fi
    
    if check_health "bars" 8083 "Bars Service"; then
        BARS_OK=true
    fi
    
    if [ "$INGEST_OK" = true ] && [ "$BARS_OK" = true ]; then
        return 0
    else
        return 1
    fi
}

# Check data flow
check_data_flow() {
    echo -e "\n${YELLOW}Data Flow:${NC}"
    
    # Check ticks stream
    if command -v docker &> /dev/null; then
        TICKS=$(docker exec stock-scanner-redis redis-cli XLEN ticks 2>/dev/null || echo "0")
        echo -e "  Ticks in stream: ${BLUE}$TICKS${NC}"
        
        BARS=$(docker exec stock-scanner-redis redis-cli XLEN bars.finalized 2>/dev/null || echo "0")
        echo -e "  Finalized bars: ${BLUE}$BARS${NC}"
        
        if [ "$TICKS" -gt 0 ]; then
            echo -e "  ${GREEN}✓ Ticks are being published${NC}"
        else
            echo -e "  ${YELLOW}⚠ No ticks yet (waiting for batch flush)${NC}"
        fi
    fi
}

# Main
main() {
    check_infrastructure
    check_go_services
    check_data_flow
    
    echo -e "\n${BLUE}Service URLs:${NC}"
    echo -e "  Ingest Health: http://localhost:8081/health"
    echo -e "  Ingest Metrics: http://localhost:8081/metrics"
    echo -e "  Bars Health: http://localhost:8083/health"
    echo -e "  Bars Metrics: http://localhost:8083/metrics"
    echo -e "  Prometheus: http://localhost:9090"
    echo -e "  Grafana: http://localhost:3000"
}

main

