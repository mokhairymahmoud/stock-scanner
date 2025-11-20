-- Migration: Create toplist_configs table
-- Description: Stores user-custom toplist configurations and system toplist metadata
-- Created: 2024-01-01

CREATE TABLE IF NOT EXISTS toplist_configs (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255), -- NULL for system toplists
    name VARCHAR(255) NOT NULL,
    description TEXT,
    metric VARCHAR(50) NOT NULL, -- change_pct, volume, rsi, relative_volume, vwap_dist
    time_window VARCHAR(10) NOT NULL, -- 1m, 5m, 15m, 1h, 1d
    sort_order VARCHAR(10) NOT NULL, -- asc, desc
    filters JSONB, -- Filtering criteria (min_volume, price_min, price_max, etc.)
    columns JSONB, -- Display columns configuration
    color_scheme JSONB, -- Color coding configuration
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_toplist_configs_user_id ON toplist_configs(user_id);
CREATE INDEX IF NOT EXISTS idx_toplist_configs_enabled ON toplist_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_toplist_configs_created_at ON toplist_configs(created_at);
CREATE INDEX IF NOT EXISTS idx_toplist_configs_user_enabled ON toplist_configs(user_id, enabled) WHERE user_id IS NOT NULL;

-- Add comments for documentation
COMMENT ON TABLE toplist_configs IS 'Stores toplist configurations for both system and user-custom toplists';
COMMENT ON COLUMN toplist_configs.id IS 'Unique identifier for the toplist';
COMMENT ON COLUMN toplist_configs.user_id IS 'User ID for user-custom toplists, NULL for system toplists';
COMMENT ON COLUMN toplist_configs.metric IS 'Metric type used for ranking (change_pct, volume, rsi, relative_volume, vwap_dist)';
COMMENT ON COLUMN toplist_configs.time_window IS 'Time window for metric calculation (1m, 5m, 15m, 1h, 1d)';
COMMENT ON COLUMN toplist_configs.sort_order IS 'Sort order for rankings (asc for ascending, desc for descending)';
COMMENT ON COLUMN toplist_configs.filters IS 'JSON object containing filter criteria (min_volume, price_min, price_max, exchange, etc.)';
COMMENT ON COLUMN toplist_configs.columns IS 'JSON array of column names to display in the toplist';
COMMENT ON COLUMN toplist_configs.color_scheme IS 'JSON object containing color coding configuration';

