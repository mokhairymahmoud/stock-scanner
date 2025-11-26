-- Remove cooldown column from rules table
-- Cooldown is now global and configured via SCANNER_COOLDOWN_DEFAULT env var

ALTER TABLE rules DROP COLUMN IF EXISTS cooldown;

-- Add comment
COMMENT ON TABLE rules IS 'Stores trading rules that can be used by the scanner. Cooldown is now global and configured via SCANNER_COOLDOWN_DEFAULT env var.';

