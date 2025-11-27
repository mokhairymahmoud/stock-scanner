#!/bin/bash

# API Examples Script
# This script demonstrates how to create Rules and Toplists using the API

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"

echo "=========================================="
echo "Stock Scanner API Examples"
echo "=========================================="
echo ""

# Example 1: Create a Rule using RSI Indicator
echo "Example 1: Creating Rule - RSI Oversold"
echo "----------------------------------------"
curl -X POST "${API_BASE_URL}/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "RSI Oversold Alert",
    "description": "Alert when RSI(14) is below 30 (oversold)",
    "conditions": [
      {
        "metric": "rsi_14",
        "operator": "<",
        "value": 30
      }
    ],
    "enabled": true
  }' | jq '.'
echo ""
echo ""

# Example 2: Create a Rule using Multiple Indicators
echo "Example 2: Creating Rule - RSI Overbought with Momentum"
echo "--------------------------------------------------------"
curl -X POST "${API_BASE_URL}/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Overbought with Momentum",
    "description": "Alert when RSI > 70, price up 5%, and volume spike",
    "conditions": [
      {
        "metric": "rsi_14",
        "operator": ">",
        "value": 70
      },
      {
        "metric": "price_change_5m_pct",
        "operator": ">",
        "value": 5.0
      }
    ],
    "enabled": true
  }' | jq '.'
echo ""
echo ""

# Example 3: Create a Toplist for RSI Extremes
echo "Example 3: Creating Toplist - RSI Overbought Stocks"
echo "---------------------------------------------------"
curl -X POST "${API_BASE_URL}/api/v1/toplists/user" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "RSI Overbought Stocks",
    "description": "Top stocks with highest RSI values (overbought)",
    "metric": "rsi",
    "time_window": "1m",
    "sort_order": "desc",
    "filters": {
      "min_volume": 1000000,
      "price_min": 5.0,
      "price_max": 500.0
    },
    "columns": ["symbol", "rsi_14", "price", "volume", "change_pct"],
    "enabled": true
  }' | jq '.'
echo ""
echo ""

# Example 4: Create a Toplist for Price Change Leaders
echo "Example 4: Creating Toplist - 5-Minute Gainers"
echo "-----------------------------------------------"
curl -X POST "${API_BASE_URL}/api/v1/toplists/user" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "5-Minute Gainers",
    "description": "Stocks with highest 5-minute price change",
    "metric": "change_pct",
    "time_window": "5m",
    "sort_order": "desc",
    "filters": {
      "min_volume": 500000,
      "min_change_pct": 1.0
    },
    "columns": ["symbol", "price", "change_pct", "volume", "rsi_14"],
    "enabled": true
  }' | jq '.'
echo ""
echo ""

# Example 5: Create a Rule using ATR for Volatility
echo "Example 5: Creating Rule - High Volatility"
echo "------------------------------------------"
curl -X POST "${API_BASE_URL}/api/v1/rules" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Volatility Alert",
    "description": "Alert when ATR(14) indicates high volatility",
    "conditions": [
      {
        "metric": "atr_14",
        "operator": ">",
        "value": 2.0
      }
    ],
    "enabled": true
  }' | jq '.'
echo ""
echo ""

# Example 6: Create a Toplist for Volume Leaders
echo "Example 6: Creating Toplist - Volume Leaders"
echo "--------------------------------------------"
curl -X POST "${API_BASE_URL}/api/v1/toplists/user" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Volume Leaders",
    "description": "Stocks with highest trading volume",
    "metric": "volume",
    "time_window": "1m",
    "sort_order": "desc",
    "filters": {
      "min_volume": 2000000
    },
    "columns": ["symbol", "volume", "price", "change_pct"],
    "enabled": true
  }' | jq '.'
echo ""
echo ""

echo "=========================================="
echo "Examples completed!"
echo "=========================================="
echo ""
echo "To query toplist rankings, use:"
echo "  curl ${API_BASE_URL}/api/v1/toplists/user/{toplist_id}/rankings"
echo ""
echo "To list all rules, use:"
echo "  curl ${API_BASE_URL}/api/v1/rules"
echo ""

