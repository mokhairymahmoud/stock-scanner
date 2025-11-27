package metrics

import (
	"testing"
	"time"
)

func TestMinutesInMarketComputer(t *testing.T) {
	tests := []struct {
		name        string
		lastTickTime time.Time
		lastUpdate   time.Time
		wantValue   float64
		wantOk      bool
	}{
		{
			name:        "market open - 30 minutes after open",
			lastTickTime: createMarketTime(9, 30+30, 0), // 10:00 AM ET
			wantValue:   30.0,
			wantOk:      true,
		},
		{
			name:        "market open - 1 hour after open",
			lastTickTime: createMarketTime(9, 30+60, 0), // 10:30 AM ET
			wantValue:   60.0,
			wantOk:      true,
		},
		{
			name:        "premarket - before market open",
			lastTickTime: createMarketTime(8, 0, 0), // 8:00 AM ET
			wantValue:   0,
			wantOk:      false,
		},
		{
			name:        "postmarket - after market close",
			lastTickTime: createMarketTime(16, 30, 0), // 4:30 PM ET
			wantValue:   420.0, // 7 hours = 420 minutes (9:30 AM to 4:30 PM)
			wantOk:      true,
		},
		{
			name:        "weekend - market closed",
			lastTickTime: createWeekendTime(),
			wantValue:   0,
			wantOk:      false,
		},
		{
			name:        "use last update if no tick time",
			lastUpdate:   createMarketTime(10, 0, 0), // 10:00 AM ET
			wantValue:   30.0,
			wantOk:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			computer := &MinutesInMarketComputer{}
			snapshot := &SymbolStateSnapshot{
				LastTickTime: tt.lastTickTime,
				LastUpdate:   tt.lastUpdate,
			}

			value, ok := computer.Compute(snapshot)
			if ok != tt.wantOk {
				t.Errorf("Compute() ok = %v, want %v", ok, tt.wantOk)
			}
			if ok {
				// Allow small floating point differences
				if abs(value-tt.wantValue) > 1.0 {
					t.Errorf("Compute() value = %v, want %v", value, tt.wantValue)
				}
			}
		})
	}
}

func TestMinutesSinceNewsComputer(t *testing.T) {
	computer := &MinutesSinceNewsComputer{}
	snapshot := &SymbolStateSnapshot{}

	// Should return false until news data integration is implemented
	value, ok := computer.Compute(snapshot)
	if ok {
		t.Errorf("Compute() ok = true, want false (news data not integrated)")
	}
	if value != 0 {
		t.Errorf("Compute() value = %v, want 0", value)
	}
}

func TestHoursSinceNewsComputer(t *testing.T) {
	computer := &HoursSinceNewsComputer{}
	snapshot := &SymbolStateSnapshot{}

	// Should return false until news data integration is implemented
	value, ok := computer.Compute(snapshot)
	if ok {
		t.Errorf("Compute() ok = true, want false (news data not integrated)")
	}
	if value != 0 {
		t.Errorf("Compute() value = %v, want 0", value)
	}
}

func TestDaysSinceNewsComputer(t *testing.T) {
	computer := &DaysSinceNewsComputer{}
	snapshot := &SymbolStateSnapshot{}

	// Should return false until news data integration is implemented
	value, ok := computer.Compute(snapshot)
	if ok {
		t.Errorf("Compute() ok = true, want false (news data not integrated)")
	}
	if value != 0 {
		t.Errorf("Compute() value = %v, want 0", value)
	}
}

func TestDaysUntilEarningsComputer(t *testing.T) {
	computer := &DaysUntilEarningsComputer{}
	snapshot := &SymbolStateSnapshot{}

	// Should return false until earnings data integration is implemented
	value, ok := computer.Compute(snapshot)
	if ok {
		t.Errorf("Compute() ok = true, want false (earnings data not integrated)")
	}
	if value != 0 {
		t.Errorf("Compute() value = %v, want 0", value)
	}
}

// Helper functions

func createMarketTime(hour, minute, second int) time.Time {
	// Create a time on a weekday (e.g., Tuesday) in Eastern Time
	etLocation, _ := time.LoadLocation("America/New_York")
	// Use a known Tuesday (e.g., 2024-01-02)
	date := time.Date(2024, 1, 2, hour, minute, second, 0, etLocation)
	return date
}

func createWeekendTime() time.Time {
	// Create a time on a weekend (e.g., Saturday)
	etLocation, _ := time.LoadLocation("America/New_York")
	// Use a known Saturday (e.g., 2024-01-06)
	date := time.Date(2024, 1, 6, 10, 0, 0, 0, etLocation)
	return date
}

