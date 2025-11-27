package metrics

import (
	"time"
)

// Time-based filter computers implement time-related calculations

// MinutesInMarketComputer computes minutes since market open (9:30 AM ET)
// Metric name: minutes_in_market
// Returns 0 if market is not open (premarket, postmarket, closed)
type MinutesInMarketComputer struct{}

func (c *MinutesInMarketComputer) Name() string { return "minutes_in_market" }

func (c *MinutesInMarketComputer) Dependencies() []string { return nil }

func (c *MinutesInMarketComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// Use current time or last tick time
	var t time.Time
	if !snapshot.LastTickTime.IsZero() {
		t = snapshot.LastTickTime
	} else if !snapshot.LastUpdate.IsZero() {
		t = snapshot.LastUpdate
	} else {
		t = time.Now()
	}

	// Calculate minutes since market open (9:30 AM ET)
	minutes := minutesSinceMarketOpen(t)
	if minutes <= 0 {
		// Market not open yet or closed
		return 0, false
	}

	return float64(minutes), true
}

// minutesSinceMarketOpen calculates minutes since market open (9:30 AM ET)
// Duplicated from scanner package to avoid import cycle
func minutesSinceMarketOpen(t time.Time) int {
	etLocation, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback: assume EST (UTC-5)
		year, month, day := t.UTC().Date()
		marketOpen := time.Date(year, month, day, 14, 30, 0, 0, time.UTC)
		if t.Before(marketOpen) {
			return 0
		}
		minutes := int(t.Sub(marketOpen).Minutes())
		if minutes < 0 {
			return 0
		}
		return minutes
	}

	// Get today's market open time (9:30 AM ET)
	etTime := t.In(etLocation)
	year, month, day := etTime.Date()
	marketOpen := time.Date(year, month, day, 9, 30, 0, 0, etLocation)

	// Check if it's a weekday
	weekday := etTime.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return 0
	}

	// If current time is before market open, return 0
	if t.Before(marketOpen) {
		return 0
	}

	// Calculate minutes since market open
	minutes := int(t.Sub(marketOpen).Minutes())
	if minutes < 0 {
		return 0
	}

	// Check if market is closed (after 4:00 PM ET)
	marketClose := time.Date(year, month, day, 16, 0, 0, 0, etLocation)
	if t.After(marketClose) {
		// Still return minutes since open (even if in postmarket)
		return minutes
	}

	return minutes
}

// MinutesSinceNewsComputer computes minutes since last news
// Metric name: minutes_since_news
// Note: Requires news data integration. For now, returns 0 if no news data available.
type MinutesSinceNewsComputer struct{}

func (c *MinutesSinceNewsComputer) Name() string { return "minutes_since_news" }

func (c *MinutesSinceNewsComputer) Dependencies() []string { return nil }

func (c *MinutesSinceNewsComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// TODO: Integrate news data source
	// For now, this is a placeholder that returns false
	// In full implementation:
	// 1. Get last news timestamp from external data source
	// 2. Calculate minutes since that timestamp
	// 3. Return the value

	// Placeholder: check if we have news timestamp in indicators or state
	// This would need to be populated by an external data service
	return 0, false
}

// HoursSinceNewsComputer computes hours since last news
// Metric name: hours_since_news
type HoursSinceNewsComputer struct{}

func (c *HoursSinceNewsComputer) Name() string { return "hours_since_news" }

func (c *HoursSinceNewsComputer) Dependencies() []string { return nil }

func (c *HoursSinceNewsComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// TODO: Integrate news data source
	// Similar to MinutesSinceNewsComputer but returns hours
	return 0, false
}

// DaysSinceNewsComputer computes days since last news
// Metric name: days_since_news
type DaysSinceNewsComputer struct{}

func (c *DaysSinceNewsComputer) Name() string { return "days_since_news" }

func (c *DaysSinceNewsComputer) Dependencies() []string { return nil }

func (c *DaysSinceNewsComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// TODO: Integrate news data source
	// Similar to MinutesSinceNewsComputer but returns days
	return 0, false
}

// DaysUntilEarningsComputer computes days until next earnings
// Metric name: days_until_earnings
// Note: Requires earnings calendar integration. For now, returns 0 if no earnings data available.
type DaysUntilEarningsComputer struct{}

func (c *DaysUntilEarningsComputer) Name() string { return "days_until_earnings" }

func (c *DaysUntilEarningsComputer) Dependencies() []string { return nil }

func (c *DaysUntilEarningsComputer) Compute(snapshot *SymbolStateSnapshot) (float64, bool) {
	// TODO: Integrate earnings calendar data source
	// For now, this is a placeholder that returns false
	// In full implementation:
	// 1. Get next earnings date from external data source
	// 2. Calculate days until that date
	// 3. Return the value

	// Placeholder: check if we have earnings date in indicators or state
	// This would need to be populated by an external data service
	return 0, false
}

