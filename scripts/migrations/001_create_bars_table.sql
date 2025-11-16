-- Create bars table with TimescaleDB hypertable
CREATE TABLE IF NOT EXISTS bars_1m (
    symbol VARCHAR(10) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    open DOUBLE PRECISION NOT NULL,
    high DOUBLE PRECISION NOT NULL,
    low DOUBLE PRECISION NOT NULL,
    close DOUBLE PRECISION NOT NULL,
    volume BIGINT NOT NULL,
    vwap DOUBLE PRECISION NOT NULL,
    PRIMARY KEY (symbol, timestamp)
);

-- Create hypertable (TimescaleDB extension)
SELECT create_hypertable('bars_1m', 'timestamp', if_not_exists => TRUE);

-- Create index on symbol for faster lookups
CREATE INDEX IF NOT EXISTS idx_bars_1m_symbol ON bars_1m (symbol);

-- Create index on timestamp for time-based queries
CREATE INDEX IF NOT EXISTS idx_bars_1m_timestamp ON bars_1m (timestamp DESC);

-- Create composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_bars_1m_symbol_timestamp ON bars_1m (symbol, timestamp DESC);

