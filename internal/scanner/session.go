package scanner

import (
	"time"
)

// MarketSession represents the current market session
type MarketSession string

const (
	SessionPreMarket MarketSession = "premarket"
	SessionMarket    MarketSession = "market"
	SessionPostMarket MarketSession = "postmarket"
	SessionClosed    MarketSession = "closed"
)

// GetMarketSession determines the current market session based on the given time
// Uses Eastern Time (ET) which is UTC-5 (EST) or UTC-4 (EDT)
// Market hours:
// - Pre-Market: 4:00 AM - 9:30 AM ET
// - Market: 9:30 AM - 4:00 PM ET
// - Post-Market: 4:00 PM - 8:00 PM ET
func GetMarketSession(t time.Time) MarketSession {
	// Load Eastern Time location
	// Note: time.LoadLocation may fail if tzdata is not available
	// In that case, we'll use a fallback calculation
	etLocation, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback: assume EST (UTC-5) - this is approximate
		return getMarketSessionFallback(t)
	}

	// Convert to Eastern Time
	etTime := t.In(etLocation)

	// Check if it's a weekday (Monday-Friday)
	weekday := etTime.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return SessionClosed
	}

	hour := etTime.Hour()
	minute := etTime.Minute()
	timeOfDay := hour*60 + minute // Minutes since midnight

	// Pre-Market: 4:00 AM - 9:30 AM ET (240 - 570 minutes)
	if timeOfDay >= 240 && timeOfDay < 570 {
		return SessionPreMarket
	}

	// Market: 9:30 AM - 4:00 PM ET (570 - 960 minutes)
	if timeOfDay >= 570 && timeOfDay < 960 {
		return SessionMarket
	}

	// Post-Market: 4:00 PM - 8:00 PM ET (960 - 1200 minutes)
	if timeOfDay >= 960 && timeOfDay < 1200 {
		return SessionPostMarket
	}

	return SessionClosed
}

// getMarketSessionFallback is a fallback when timezone data is not available
// Assumes EST (UTC-5) - this is approximate and doesn't handle DST
func getMarketSessionFallback(t time.Time) MarketSession {
	utcTime := t.UTC()
	weekday := utcTime.Weekday()

	if weekday == time.Saturday || weekday == time.Sunday {
		return SessionClosed
	}

	hour := utcTime.Hour()
	minute := utcTime.Minute()
	timeOfDay := hour*60 + minute

	// EST offset: UTC-5, so add 5 hours to UTC time to get ET
	// Pre-Market: 4:00-9:30 ET = 9:00-14:30 UTC
	if timeOfDay >= 540 && timeOfDay < 870 {
		return SessionPreMarket
	}

	// Market: 9:30-16:00 ET = 14:30-21:00 UTC
	if timeOfDay >= 870 && timeOfDay < 1260 {
		return SessionMarket
	}

	// Post-Market: 16:00-20:00 ET = 21:00-01:00 UTC (next day)
	if timeOfDay >= 1260 || timeOfDay < 60 {
		return SessionPostMarket
	}

	return SessionClosed
}

// IsMarketOpen returns true if the market is currently open (Market session)
func IsMarketOpen(t time.Time) bool {
	return GetMarketSession(t) == SessionMarket
}

// IsPreMarket returns true if it's pre-market hours
func IsPreMarket(t time.Time) bool {
	return GetMarketSession(t) == SessionPreMarket
}

// IsPostMarket returns true if it's post-market hours
func IsPostMarket(t time.Time) bool {
	return GetMarketSession(t) == SessionPostMarket
}

// GetMarketOpenTime returns the market open time (9:30 AM ET) for the given date
func GetMarketOpenTime(date time.Time) time.Time {
	etLocation, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback: assume EST (UTC-5)
		year, month, day := date.UTC().Date()
		return time.Date(year, month, day, 14, 30, 0, 0, time.UTC)
	}

	// Get the date in ET
	etDate := date.In(etLocation)
	year, month, day := etDate.Date()

	// Create 9:30 AM ET
	marketOpen := time.Date(year, month, day, 9, 30, 0, 0, etLocation)
	return marketOpen
}

// GetMarketCloseTime returns the market close time (4:00 PM ET) for the given date
func GetMarketCloseTime(date time.Time) time.Time {
	etLocation, err := time.LoadLocation("America/New_York")
	if err != nil {
		// Fallback: assume EST (UTC-5)
		year, month, day := date.UTC().Date()
		return time.Date(year, month, day, 21, 0, 0, 0, time.UTC)
	}

	// Get the date in ET
	etDate := date.In(etLocation)
	year, month, day := etDate.Date()

	// Create 4:00 PM ET
	marketClose := time.Date(year, month, day, 16, 0, 0, 0, etLocation)
	return marketClose
}

// MinutesSinceMarketOpen calculates the number of minutes since market open (9:30 AM ET)
// Returns 0 if market hasn't opened yet today, or if it's not a trading day
func MinutesSinceMarketOpen(t time.Time) int {
	session := GetMarketSession(t)
	if session == SessionClosed {
		return 0
	}

	// Get today's market open time
	marketOpen := GetMarketOpenTime(t)

	// If current time is before market open, return 0
	if t.Before(marketOpen) {
		return 0
	}

	// Calculate minutes since market open
	minutes := int(t.Sub(marketOpen).Minutes())
	if minutes < 0 {
		return 0
	}

	return minutes
}

