# RedisInsight Connection Guide

## Access RedisInsight

- **URL**: http://localhost:8001

## Connecting to Redis from RedisInsight

When adding a Redis database connection in RedisInsight, use these settings:

### Option 1: Connect from Browser (Recommended)

Since RedisInsight runs in Docker but you access it from your browser, use:

- **Host**: `localhost` (or `127.0.0.1`)
- **Port**: `6379`
- **Database Alias**: `Stock Scanner Redis` (or any name)
- **Username**: (leave empty)
- **Password**: (leave empty)

### Option 2: Connect from RedisInsight Container

If RedisInsight needs to connect internally (rare), use:

- **Host**: `redis` (Docker service name)
- **Port**: `6379`
- **Username**: (leave empty)
- **Password**: (leave empty)

## Troubleshooting

### "Could not connect to redis:6379"

1. **Verify Redis is running**:
   ```bash
   docker ps | grep redis
   ```

2. **Test Redis connection**:
   ```bash
   docker exec stock-scanner-redis redis-cli ping
   # Should return: PONG
   ```

3. **Check Redis is accessible from host**:
   ```bash
   redis-cli -h localhost -p 6379 ping
   # Or if redis-cli is not installed:
   nc -zv localhost 6379
   ```

4. **Verify port mapping**:
   ```bash
   docker ps | grep redis
   # Should show: 0.0.0.0:6379->6379/tcp
   ```

5. **Check Redis bind configuration**:
   - Redis should be bound to `0.0.0.0` (all interfaces) to accept connections
   - Check with: `docker exec stock-scanner-redis redis-cli CONFIG GET bind`
   - Should return: `bind` and `*` or empty (which means all interfaces)

### Connection Refused

If you get "Connection refused":

1. **Restart Redis container**:
   ```bash
   docker-compose -f config/docker-compose.yaml restart redis
   ```

2. **Check Redis logs**:
   ```bash
   docker-compose -f config/docker-compose.yaml logs redis
   ```

3. **Verify network**:
   ```bash
   docker network inspect stock-scanner_stock-scanner-network | grep -A 5 redis
   ```

### Still Can't Connect?

1. **Try connecting from command line first**:
   ```bash
   # From host machine
   redis-cli -h localhost -p 6379 ping
   
   # Or using Docker
   docker exec -it stock-scanner-redis redis-cli ping
   ```

2. **Check firewall settings** (if on Linux):
   ```bash
   sudo ufw status
   ```

3. **Verify Docker port mapping**:
   ```bash
   docker port stock-scanner-redis
   # Should show: 6379/tcp -> 0.0.0.0:6379
   ```

## Quick Test

Run this to verify everything is set up correctly:

```bash
# Test Redis from host
docker exec stock-scanner-redis redis-cli ping

# Test Redis from host network
nc -zv localhost 6379

# Test RedisInsight is accessible
curl http://localhost:8001/api/health
```

## Common Issues

### Issue: RedisInsight shows port 5540 instead of 8001

**Solution**: This is normal. RedisInsight v2+ uses port 5540 internally. The docker-compose maps it to 8001 on the host. Access it at http://localhost:8001.

### Issue: Can connect via redis-cli but not from RedisInsight

**Solution**: 
- Make sure you're using `localhost` (not `redis`) when connecting from the browser
- Check that RedisInsight container is on the same network
- Try restarting both containers

### Issue: Connection works but can't see data

**Solution**:
- Make sure services are running and publishing data
- Check that you're looking at the correct database (default is 0)
- Verify streams exist: `docker exec stock-scanner-redis redis-cli XLEN ticks`

