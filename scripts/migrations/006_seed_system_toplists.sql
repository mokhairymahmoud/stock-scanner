-- Migration: Seed system toplists
-- Description: Creates default system toplists in the database
-- These toplists are available to all users (user_id = NULL)
-- Created: 2024-01-01

-- Insert system toplists for price change (gainers/losers)
INSERT INTO toplist_configs (id, user_id, name, description, metric, time_window, sort_order, enabled, created_at, updated_at)
VALUES
  -- Gainers (descending order - highest change first)
  ('gainers_1m', NULL, 'Top Gainers (1m)', 'Stocks with highest 1-minute price change', 'change_pct', '1m', 'desc', true, NOW(), NOW()),
  ('gainers_5m', NULL, 'Top Gainers (5m)', 'Stocks with highest 5-minute price change', 'change_pct', '5m', 'desc', true, NOW(), NOW()),
  ('gainers_15m', NULL, 'Top Gainers (15m)', 'Stocks with highest 15-minute price change', 'change_pct', '15m', 'desc', true, NOW(), NOW()),
  ('gainers_1h', NULL, 'Top Gainers (1h)', 'Stocks with highest 1-hour price change', 'change_pct', '1h', 'desc', true, NOW(), NOW()),
  ('gainers_1d', NULL, 'Top Gainers (1d)', 'Stocks with highest daily price change', 'change_pct', '1d', 'desc', true, NOW(), NOW()),
  
  -- Losers (ascending order - lowest change first, which are the biggest losers)
  ('losers_1m', NULL, 'Top Losers (1m)', 'Stocks with lowest 1-minute price change', 'change_pct', '1m', 'asc', true, NOW(), NOW()),
  ('losers_5m', NULL, 'Top Losers (5m)', 'Stocks with lowest 5-minute price change', 'change_pct', '5m', 'asc', true, NOW(), NOW()),
  ('losers_15m', NULL, 'Top Losers (15m)', 'Stocks with lowest 15-minute price change', 'change_pct', '15m', 'asc', true, NOW(), NOW()),
  ('losers_1h', NULL, 'Top Losers (1h)', 'Stocks with lowest 1-hour price change', 'change_pct', '1h', 'asc', true, NOW(), NOW()),
  ('losers_1d', NULL, 'Top Losers (1d)', 'Stocks with lowest daily price change', 'change_pct', '1d', 'asc', true, NOW(), NOW()),
  
  -- Volume Leaders (descending order - highest volume first)
  ('volume_1m', NULL, 'Volume Leaders (1m)', 'Stocks with highest 1-minute trading volume', 'volume', '1m', 'desc', true, NOW(), NOW()),
  ('volume_5m', NULL, 'Volume Leaders (5m)', 'Stocks with highest 5-minute trading volume', 'volume', '5m', 'desc', true, NOW(), NOW()),
  ('volume_15m', NULL, 'Volume Leaders (15m)', 'Stocks with highest 15-minute trading volume', 'volume', '15m', 'desc', true, NOW(), NOW()),
  ('volume_1h', NULL, 'Volume Leaders (1h)', 'Stocks with highest 1-hour trading volume', 'volume', '1h', 'desc', true, NOW(), NOW()),
  ('volume_1d', NULL, 'Volume Leaders (1d)', 'Stocks with highest daily trading volume', 'volume', '1d', 'desc', true, NOW(), NOW()),
  
  -- RSI Extremes
  ('rsi_high', NULL, 'RSI High', 'Stocks with highest RSI values (potentially overbought)', 'rsi', '1m', 'desc', true, NOW(), NOW()),
  ('rsi_low', NULL, 'RSI Low', 'Stocks with lowest RSI values (potentially oversold)', 'rsi', '1m', 'asc', true, NOW(), NOW()),
  
  -- Relative Volume
  ('relative_volume', NULL, 'Relative Volume Leaders', 'Stocks with highest relative volume ratios', 'relative_volume', '15m', 'desc', true, NOW(), NOW()),
  
  -- VWAP Distance
  ('vwap_dist_high', NULL, 'VWAP Distance High', 'Stocks furthest above VWAP', 'vwap_dist', '5m', 'desc', true, NOW(), NOW()),
  ('vwap_dist_low', NULL, 'VWAP Distance Low', 'Stocks furthest below VWAP', 'vwap_dist', '5m', 'asc', true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Add comment
COMMENT ON TABLE toplist_configs IS 'Stores toplist configurations. System toplists have user_id = NULL and are available to all users.';

