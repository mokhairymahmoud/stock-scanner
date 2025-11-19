#!/bin/bash
# Deployment script for stock scanner services

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=========================================="
echo "Stock Scanner Deployment"
echo "==========================================${NC}"

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}Creating .env from example...${NC}"
    cp config/env.example .env
    echo -e "${YELLOW}⚠ Please edit .env with your configuration before continuing${NC}"
    echo -e "${YELLOW}For testing with mock provider, set:${NC}"
    echo -e "  MARKET_DATA_PROVIDER=mock"
    echo -e "  MARKET_DATA_API_KEY=test-key"
    echo -e "  MARKET_DATA_SYMBOLS=AAPL,MSFT,GOOGL"
    read -p "Press Enter to continue after editing .env..."
fi

# Step 1: Start Infrastructure
echo -e "\n${YELLOW}Step 1: Starting infrastructure...${NC}"
docker-compose -f config/docker-compose.yaml up -d redis timescaledb prometheus grafana redisinsight

echo -e "${YELLOW}Waiting for infrastructure to be ready...${NC}"
sleep 15

# Step 2: Run Migrations
echo -e "\n${YELLOW}Step 2: Running database migrations...${NC}"
if docker exec stock-scanner-timescaledb pg_isready -U postgres > /dev/null 2>&1; then
    docker exec -i stock-scanner-timescaledb psql -U postgres -d stock_scanner < scripts/migrations/001_create_bars_table.sql 2>&1 | grep -v "NOTICE:" | grep -v "WARNING:" || true
    echo -e "${GREEN}✓ Migrations completed${NC}"
else
    echo -e "${YELLOW}⚠ TimescaleDB not ready yet, migrations will run automatically on startup${NC}"
fi

# Step 3: Build Services
echo -e "\n${YELLOW}Step 3: Building service images...${NC}"
docker-compose -f config/docker-compose.yaml build ingest bars

# Step 4: Start Services
echo -e "\n${YELLOW}Step 4: Starting services...${NC}"
docker-compose -f config/docker-compose.yaml up -d ingest bars

# Step 5: Wait for Services
echo -e "\n${YELLOW}Step 5: Waiting for services to be ready...${NC}"
sleep 20

# Step 6: Health Checks
echo -e "\n${YELLOW}Step 6: Checking service health...${NC}"

INGEST_HEALTHY=false
BARS_HEALTHY=false

for i in {1..10}; do
    if curl -s -f http://localhost:8081/health > /dev/null 2>&1; then
        INGEST_HEALTHY=true
        echo -e "${GREEN}✓ Ingest service is healthy${NC}"
        break
    fi
    echo -e "${YELLOW}Waiting for ingest service... ($i/10)${NC}"
    sleep 2
done

for i in {1..10}; do
    if curl -s -f http://localhost:8083/health > /dev/null 2>&1; then
        BARS_HEALTHY=true
        echo -e "${GREEN}✓ Bars service is healthy${NC}"
        break
    fi
    echo -e "${YELLOW}Waiting for bars service... ($i/10)${NC}"
    sleep 2
done

# Summary
echo -e "\n${BLUE}=========================================="
echo "Deployment Summary"
echo "==========================================${NC}"

if [ "$INGEST_HEALTHY" = true ]; then
    echo -e "${GREEN}✓ Ingest Service: DEPLOYED${NC}"
    echo -e "  Health: http://localhost:8081/health"
    echo -e "  Metrics: http://localhost:8081/metrics"
else
    echo -e "${RED}✗ Ingest Service: FAILED${NC}"
    echo -e "  Check logs: docker-compose -f config/docker-compose.yaml logs ingest"
fi

if [ "$BARS_HEALTHY" = true ]; then
    echo -e "${GREEN}✓ Bars Service: DEPLOYED${NC}"
    echo -e "  Health: http://localhost:8083/health"
    echo -e "  Metrics: http://localhost:8083/metrics"
else
    echo -e "${RED}✗ Bars Service: FAILED${NC}"
    echo -e "  Check logs: docker-compose -f config/docker-compose.yaml logs bars"
fi

echo -e "\n${BLUE}Infrastructure:${NC}"
echo -e "  Redis: http://localhost:6379"
echo -e "  TimescaleDB: localhost:5432"
echo -e "  Prometheus: http://localhost:9090"
echo -e "  Grafana: http://localhost:3000 (admin/admin)"

echo -e "\n${YELLOW}Next Steps:${NC}"
echo -e "  1. Run test script: ./scripts/test_services.sh"
echo -e "  2. Check logs: make docker-logs"
echo -e "  3. View metrics in Prometheus: http://localhost:9090"

if [ "$INGEST_HEALTHY" = true ] && [ "$BARS_HEALTHY" = true ]; then
    echo -e "\n${GREEN}Deployment successful!${NC}"
    exit 0
else
    echo -e "\n${RED}Deployment completed with issues. Check logs for details.${NC}"
    exit 1
fi

