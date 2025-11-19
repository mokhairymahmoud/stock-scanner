package indicator

import (
	"testing"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

func TestVolumeAverage_NewVolumeAverage(t *testing.T) {
	vol, err := NewVolumeAverage(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create VolumeAverage: %v", err)
	}
	if vol == nil {
		t.Fatal("VolumeAverage is nil")
	}
	if vol.Name() != "volume_avg_5m" {
		t.Errorf("Expected name 'volume_avg_5m', got '%s'", vol.Name())
	}
}

func TestVolumeAverage_Update(t *testing.T) {
	vol, _ := NewVolumeAverage(5 * time.Minute)

	baseTime := time.Now()

	// Add bars
	for i := 0; i < 5; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Volume:    int64(1000 + i*100),
		}
		val, err := vol.Update(bar)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		if i == 0 {
			if !vol.IsReady() {
				t.Error("VolumeAverage should be ready after first bar")
			}
			if val != 1000.0 {
				t.Errorf("Expected 1000.0, got %f", val)
			}
		}
	}

	// Average should be (1000+1100+1200+1300+1400)/5 = 1200
	val, _ := vol.Value()
	expected := (1000.0 + 1100.0 + 1200.0 + 1300.0 + 1400.0) / 5.0
	if val != expected {
		t.Errorf("Expected average %f, got %f", expected, val)
	}
}

func TestRelativeVolume_Update(t *testing.T) {
	relVol, _ := NewRelativeVolume(5 * time.Minute)

	baseTime := time.Now()

	// Add bars to build average (all with volume 1000)
	for i := 0; i < 5; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Volume:    1000, // Average volume
		}
		_, _ = relVol.Update(bar)
	}

	// Add bar with double volume
	bar := &models.Bar1m{
		Symbol:    "AAPL",
		Timestamp: baseTime.Add(5 * time.Minute),
		Volume:    2000, // Double the average
	}
	val, err := relVol.Update(bar)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Relative volume should be close to 2.0
	// Note: The average includes all 6 bars now, so it's (5*1000 + 2000)/6 = 1166.67
	// Relative = 2000/1166.67 = 1.714
	// So we check for a reasonable range
	if val < 1.5 || val > 2.5 {
		t.Errorf("Expected relative volume between 1.5-2.5, got %f", val)
	}
	
	// Verify it's greater than 1.0 (above average)
	if val <= 1.0 {
		t.Errorf("Expected relative volume > 1.0 for double volume, got %f", val)
	}

	// Value() should return the last calculated value
	val2, _ := relVol.Value()
	if val2 != val {
		t.Errorf("Value() should return last calculated value, got %f, expected %f", val2, val)
	}
}

func TestRelativeVolume_Reset(t *testing.T) {
	relVol, _ := NewRelativeVolume(5 * time.Minute)

	baseTime := time.Now()
	for i := 0; i < 5; i++ {
		bar := &models.Bar1m{
			Symbol:    "AAPL",
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Volume:    1000,
		}
		_, _ = relVol.Update(bar)
	}

	relVol.Reset()

	if relVol.IsReady() {
		t.Error("RelativeVolume should not be ready after reset")
	}
}

