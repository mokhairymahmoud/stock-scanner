-- Create rules table for storing trading rules
-- This table stores all rules that can be used by the scanner

CREATE TABLE IF NOT EXISTS rules (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    conditions JSONB NOT NULL,
    cooldown INTEGER NOT NULL DEFAULT 10, -- Cooldown in seconds
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_rules_enabled ON rules(enabled) WHERE enabled = true;
CREATE INDEX IF NOT EXISTS idx_rules_created_at ON rules(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_rules_updated_at ON rules(updated_at DESC);

-- Add comment
COMMENT ON TABLE rules IS 'Stores trading rules that can be used by the scanner';

