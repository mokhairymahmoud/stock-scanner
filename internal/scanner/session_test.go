package scanner

import (
	"testing"
	"time"
)

func TestGetMarketSession(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string // Format: "2006-01-02 15:04:05"
		expected MarketSession
	}{
		// Pre-Market: 4:00 AM - 9:30 AM ET
		{"Pre-Market early", "2024-01-15 09:00:00", SessionPreMarket}, // 4:00 AM ET
		{"Pre-Market mid", "2024-01-15 12:00:00", SessionPreMarket},   // 7:00 AM ET
		{"Pre-Market late", "2024-01-15 14:29:00", SessionPreMarket},  // 9:29 AM ET

		// Market: 9:30 AM - 4:00 PM ET
		{"Market open", "2024-01-15 14:30:00", SessionMarket},   // 9:30 AM ET
		{"Market mid", "2024-01-15 18:00:00", SessionMarket},   // 1:00 PM ET
		{"Market late", "2024-01-15 20:59:00", SessionMarket},   // 3:59 PM ET

		// Post-Market: 4:00 PM - 8:00 PM ET
		{"Post-Market early", "2024-01-15 21:00:00", SessionPostMarket}, // 4:00 PM ET
		{"Post-Market mid", "2024-01-15 23:00:00", SessionPostMarket},   // 7:00 PM ET
		{"Post-Market late", "2024-01-16 00:59:00", SessionPostMarket},  // 7:59 PM ET (next day UTC)

		// Closed hours
		{"After hours", "2024-01-16 01:00:00", SessionClosed}, // 8:00 PM ET
		{"Before premarket", "2024-01-15 08:59:00", SessionClosed}, // 3:59 AM ET

		// Weekend
		{"Saturday", "2024-01-13 18:00:00", SessionClosed}, // Saturday
		{"Sunday", "2024-01-14 18:00:00", SessionClosed},   // Sunday
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse time in UTC (the test times are already in UTC)
			testTime, err := time.Parse("2006-01-02 15:04:05", tt.timeStr)
			if err != nil {
				t.Fatalf("Failed to parse time: %v", err)
			}
			testTime = testTime.UTC()

			result := GetMarketSession(testTime)
			if result != tt.expected {
				t.Errorf("GetMarketSession(%v) = %v, want %v", testTime, result, tt.expected)
			}
		})
	}
}

func TestMinutesSinceMarketOpen(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		expected int
	}{
		{"Before market open", "2024-01-15 14:29:00", 0}, // 9:29 AM ET
		{"At market open", "2024-01-15 14:30:00", 0},     // 9:30 AM ET
		{"5 minutes after open", "2024-01-15 14:35:00", 5}, // 9:35 AM ET
		{"1 hour after open", "2024-01-15 15:30:00", 60},   // 10:30 AM ET
		{"Weekend", "2024-01-13 18:00:00", 0},              // Saturday
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime, err := time.Parse("2006-01-02 15:04:05", tt.timeStr)
			if err != nil {
				t.Fatalf("Failed to parse time: %v", err)
			}
			testTime = testTime.UTC()

			result := MinutesSinceMarketOpen(testTime)
			if result != tt.expected {
				t.Errorf("MinutesSinceMarketOpen(%v) = %d, want %d", testTime, result, tt.expected)
			}
		})
	}
}

func TestIsMarketOpen(t *testing.T) {
	tests := []struct {
		name     string
		timeStr  string
		expected bool
	}{
		{"Market hours", "2024-01-15 18:00:00", true},   // 1:00 PM ET
		{"Pre-market", "2024-01-15 12:00:00", false},     // 7:00 AM ET
		{"Post-market", "2024-01-15 21:00:00", false},   // 4:00 PM ET
		{"Weekend", "2024-01-13 18:00:00", false},       // Saturday
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testTime, err := time.Parse("2006-01-02 15:04:05", tt.timeStr)
			if err != nil {
				t.Fatalf("Failed to parse time: %v", err)
			}
			testTime = testTime.UTC()

			result := IsMarketOpen(testTime)
			if result != tt.expected {
				t.Errorf("IsMarketOpen(%v) = %v, want %v", testTime, result, tt.expected)
			}
		})
	}
}

