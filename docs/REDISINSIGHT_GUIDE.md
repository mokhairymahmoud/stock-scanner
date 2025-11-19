# RedisInsight Guide

RedisInsight is a GUI tool for Redis that makes it easy to inspect and manage your Redis data, including streams, keys, and data structures.

## Access

- **URL**: http://localhost:8001
- **No authentication required** (for local development)

## Connecting to Redis

When you first open RedisInsight, you'll need to add a Redis connection:

1. Click **"Add Redis Database"** or **"+"** button
2. Enter connection details:
   - **Host**: `localhost` (use `localhost` when accessing from your browser, not `redis`)
   - **Port**: `6379`
   - **Database Alias**: `Stock Scanner Redis` (or any name you prefer)
   - **Username**: Leave empty (unless Redis is configured with authentication)
   - **Password**: Leave empty (unless Redis is configured with authentication)
3. Click **"Add Redis Database"**

**Important**: Since you're accessing RedisInsight from your browser (not from within Docker), you must use `localhost` as the host, not `redis`. The `redis` hostname only works from within the Docker network.

## Features

### 1. Browse Keys

- View all keys in Redis
- Filter by pattern (e.g., `livebar:*` to see all live bars)
- See key types (String, Stream, Hash, etc.)
- View TTL (Time To Live) for keys

### 2. Inspect Streams

RedisInsight provides excellent support for Redis Streams:

1. Navigate to the **"Streams"** section
2. Find your streams:
   - `ticks` - Raw tick data from ingest service
   - `bars.finalized` - Finalized 1-minute bars
3. Click on a stream to:
   - View messages in the stream
   - See message IDs and timestamps
   - Inspect message payloads (JSON)
   - Monitor stream length

### 3. View Live Bars

1. In the **"Browser"** section, search for keys matching `livebar:*`
2. Click on a key (e.g., `livebar:AAPL`)
3. View the JSON structure:
   ```json
   {
     "symbol": "AAPL",
     "timestamp": "2024-01-01T10:00:00Z",
     "open": 150.00,
     "high": 151.00,
     "low": 149.50,
     "close": 150.75,
     "volume": 1000000,
     "vwap": 150.25
   }
   ```

### 4. Monitor Stream Activity

1. Navigate to the **"Streams"** section
2. Select a stream (e.g., `ticks`)
3. Use the **"Monitor"** feature to see new messages in real-time
4. Watch as ticks are published by the ingest service

### 5. Execute Commands

Use the **"CLI"** section to run Redis commands:

```redis
# Get stream length
XLEN ticks

# Read messages from stream
XREAD COUNT 10 STREAMS ticks 0

# Get live bar for a symbol
GET livebar:AAPL

# List all live bar keys
KEYS livebar:*

# Get finalized bars count
XLEN bars.finalized
```

## Common Use Cases

### Verify Data Flow

1. **Check Ticks Stream**:
   - Navigate to Streams → `ticks`
   - Verify messages are being added
   - Check message count: `XLEN ticks`

2. **Check Live Bars**:
   - Browser → Search `livebar:*`
   - Verify keys exist for each symbol
   - Check that values are updating

3. **Check Finalized Bars**:
   - Streams → `bars.finalized`
   - Verify messages are being added at minute boundaries
   - Inspect bar data structure

### Debug Issues

1. **No Ticks in Stream**:
   - Check if ingest service is running
   - Verify Redis connection in ingest service logs
   - Check provider connection status

2. **No Live Bars**:
   - Verify bars service is consuming from `ticks` stream
   - Check bars service logs
   - Verify consumer group status

3. **No Finalized Bars**:
   - Wait for a minute boundary (bars finalize at the end of each minute)
   - Check bars service logs for finalization events
   - Verify database writes are working

## Tips

- **Refresh**: Use the refresh button to update views
- **Auto-refresh**: Enable auto-refresh for real-time monitoring
- **Filter**: Use pattern matching to find specific keys
- **Export**: Export data for analysis
- **Search**: Use the search feature to find keys quickly

## Troubleshooting

### Can't Connect to Redis

1. Verify Redis container is running:
   ```bash
   docker ps | grep redis
   ```

2. Check Redis is accessible:
   ```bash
   docker exec stock-scanner-redis redis-cli ping
   ```

3. Verify network connectivity:
   - If connecting from host: use `localhost`
   - If connecting from Docker: use `redis` (service name)

### Stream Not Showing

- Streams may not appear until they have at least one message
- Try adding a message or wait for the service to publish data
- Refresh the RedisInsight view

### Performance

- For large streams, use pagination
- Use filters to narrow down results
- Disable auto-refresh if it's slowing down the UI

