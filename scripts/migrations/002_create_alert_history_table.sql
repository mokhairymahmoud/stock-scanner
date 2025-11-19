-- Create alert_history table for storing alerts
-- This table stores all alerts that have been processed by the alert service

CREATE TABLE IF NOT EXISTS alert_history (
    id VARCHAR(255) PRIMARY KEY,
    rule_id VARCHAR(255) NOT NULL,
    rule_name VARCHAR(255) NOT NULL,
    symbol VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    price DECIMAL(20, 8) NOT NULL,
    message TEXT,
    metadata JSONB,
    trace_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_alert_history_symbol ON alert_history(symbol);
CREATE INDEX IF NOT EXISTS idx_alert_history_rule_id ON alert_history(rule_id);
CREATE INDEX IF NOT EXISTS idx_alert_history_timestamp ON alert_history(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_alert_history_symbol_timestamp ON alert_history(symbol, timestamp DESC);

-- Create hypertable for time-series data (TimescaleDB)
-- This enables efficient time-based queries and automatic data retention
SELECT create_hypertable('alert_history', 'timestamp', if_not_exists => TRUE);

-- Add comment
COMMENT ON TABLE alert_history IS 'Stores all processed alerts with deduplication and filtering applied';

