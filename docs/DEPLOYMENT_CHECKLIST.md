# Deployment Checklist

Use this checklist to verify your deployment is working correctly.

## Pre-Deployment

- [ ] `.env` file created from `config/env.example`
- [ ] Environment variables configured (especially for mock provider testing)
- [ ] Docker and Docker Compose installed
- [ ] Ports 6379, 5432, 8080-8091, 9090, 3000 available

## Deployment Steps

- [ ] Run deployment script: `make docker-deploy`
- [ ] Wait for infrastructure to start (15-20 seconds)
- [ ] Verify infrastructure services are healthy
- [ ] Verify Go services are built and started
- [ ] Wait for services to be ready (20-30 seconds)

## Post-Deployment Verification

### Infrastructure

- [ ] Redis is accessible: `docker exec stock-scanner-redis redis-cli ping`
- [ ] TimescaleDB is accessible: `docker exec stock-scanner-timescaledb pg_isready -U postgres`
- [ ] Prometheus is accessible: `curl http://localhost:9090/-/healthy`
- [ ] Grafana is accessible: `curl http://localhost:3000/api/health`

### Ingest Service

- [ ] Health check passes: `curl http://localhost:8081/health`
- [ ] Provider is connected (check health response)
- [ ] Metrics endpoint accessible: `curl http://localhost:8081/metrics`
- [ ] Ticks are being published to Redis stream (wait 10-15 seconds)
  ```bash
  docker exec -it stock-scanner-redis redis-cli XLEN ticks
  ```

### Bars Service

- [ ] Health check passes: `curl http://localhost:8083/health`
- [ ] Consumer is running (check health response)
- [ ] Publisher is running (check health response)
- [ ] Database client is running (check health response)
- [ ] Metrics endpoint accessible: `curl http://localhost:8083/metrics`
- [ ] Live bars are being published (wait 10-15 seconds)
  ```bash
  docker exec -it stock-scanner-redis redis-cli GET livebar:AAPL
  ```
- [ ] Finalized bars are being published (wait 1-2 minutes for minute boundary)
  ```bash
  docker exec -it stock-scanner-redis redis-cli XLEN bars.finalized
  ```
- [ ] Bars are being written to database (wait 1-2 minutes)
  ```bash
  docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
    -c "SELECT COUNT(*) FROM bars_1m;"
  ```

## Data Flow Verification

### End-to-End Test

1. **Start Services**
   ```bash
   make docker-deploy
   ```

2. **Wait for Initial Data** (30 seconds)
   ```bash
   sleep 30
   ```

3. **Verify Ticks Stream**
   ```bash
   docker exec -it stock-scanner-redis redis-cli XLEN ticks
   # Should show > 0
   ```

4. **Verify Bars Processing** (wait 1-2 minutes)
   ```bash
   # Check finalized bars stream
   docker exec -it stock-scanner-redis redis-cli XLEN bars.finalized
   
   # Check database
   docker exec -it stock-scanner-timescaledb psql -U postgres -d stock_scanner \
     -c "SELECT symbol, timestamp, open, close, volume FROM bars_1m ORDER BY timestamp DESC LIMIT 5;"
   ```

5. **Verify Live Bars**
   ```bash
   docker exec -it stock-scanner-redis redis-cli GET livebar:AAPL | jq .
   ```

## Monitoring Verification

- [ ] Prometheus is scraping metrics from services
- [ ] Metrics are visible in Prometheus UI
- [ ] Grafana can connect to Prometheus
- [ ] Service logs are accessible via `make docker-logs`

## Troubleshooting

If any step fails:

1. **Check Service Logs**
   ```bash
   make docker-logs-service SERVICE=ingest
   make docker-logs-service SERVICE=bars
   ```

2. **Check Service Status**
   ```bash
   docker-compose -f config/docker-compose.yaml ps
   ```

3. **Check Health Endpoints**
   ```bash
   curl http://localhost:8081/health | jq .
   curl http://localhost:8083/health | jq .
   ```

4. **Verify Environment Variables**
   ```bash
   docker exec stock-scanner-ingest env | grep MARKET_DATA
   docker exec stock-scanner-bars env | grep DB_
   ```

5. **Restart Services**
   ```bash
   docker-compose -f config/docker-compose.yaml restart ingest bars
   ```

## Success Criteria

✅ All infrastructure services running  
✅ Ingest service healthy and publishing ticks  
✅ Bars service healthy and processing ticks  
✅ Live bars being published to Redis  
✅ Finalized bars being published to Redis Stream  
✅ Bars being written to TimescaleDB  
✅ Metrics available in Prometheus  

Once all items are checked, your deployment is successful!

