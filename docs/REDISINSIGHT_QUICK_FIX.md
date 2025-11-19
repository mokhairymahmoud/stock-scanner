# RedisInsight Connection Quick Fix

## The Issue

If you see: **"Could not connect to redis:6379"**

This happens because you're trying to connect using the Docker service name `redis`, but you're accessing RedisInsight from your browser (outside Docker).

## The Solution

When adding a Redis connection in RedisInsight, use:

- **Host**: `localhost` (NOT `redis`)
- **Port**: `6379`
- **Username**: (leave empty)
- **Password**: (leave empty)

## Why?

- `redis` is the Docker service name - it only works from within the Docker network
- `localhost` works from your browser because Redis port 6379 is mapped to your host machine
- Since you access RedisInsight via http://localhost:8001 (from your browser), you must also connect to Redis using `localhost`

## Step-by-Step

1. Open RedisInsight: http://localhost:8001
2. Click **"Add Redis Database"** or **"+"**
3. Enter:
   - Host: `localhost`
   - Port: `6379`
   - Database Alias: `Stock Scanner Redis`
4. Click **"Add Redis Database"**

## Verify It Works

After connecting, you should see:
- Redis database appears in the list
- You can browse keys
- You can see streams (`ticks`, `bars.finalized`)
- You can view live bars (`livebar:*`)

## Still Having Issues?

1. **Verify Redis is running**:
   ```bash
   docker ps | grep redis
   ```

2. **Test Redis connection**:
   ```bash
   docker exec stock-scanner-redis redis-cli ping
   # Should return: PONG
   ```

3. **Check Redis port is accessible**:
   ```bash
   nc -zv localhost 6379
   # Should show: Connection to localhost port 6379 [tcp/*] succeeded!
   ```

4. **Restart RedisInsight**:
   ```bash
   docker-compose -f config/docker-compose.yaml restart redisinsight
   ```

