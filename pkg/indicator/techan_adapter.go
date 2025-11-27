package indicator

import (
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

// TechanCalculator wraps a Techan indicator to implement Calculator interface
type TechanCalculator struct {
	name      string
	series    *techan.TimeSeries
	indicator techan.Indicator
	ready     bool
	period    int
}

// NewTechanCalculator creates a new Techan-based calculator
// The indicator must be created with the same TimeSeries that will be used for updates
func NewTechanCalculator(
	name string,
	indicator techan.Indicator,
	period int,
) *TechanCalculator {
	// Create TimeSeries that will be shared with the indicator
	series := techan.NewTimeSeries()

	return &TechanCalculator{
		name:      name,
		series:    series,
		indicator: indicator,
		period:    period,
		ready:     false,
	}
}

func (t *TechanCalculator) Name() string {
	return t.name
}

func (t *TechanCalculator) Update(bar *models.Bar1m) (float64, error) {
	if bar == nil {
		return 0, fmt.Errorf("bar cannot be nil")
	}

	// Convert Bar1m to techan.Candle
	timePeriod := techan.NewTimePeriod(bar.Timestamp, time.Minute)
	candle := techan.NewCandle(timePeriod)

	candle.OpenPrice = big.NewDecimal(bar.Open)
	candle.MaxPrice = big.NewDecimal(bar.High)
	candle.MinPrice = big.NewDecimal(bar.Low)
	candle.ClosePrice = big.NewDecimal(bar.Close)
	candle.Volume = big.NewDecimal(float64(bar.Volume))

	t.series.AddCandle(candle)

	// Check if we have enough data
	lastIndex := t.series.LastIndex()
	if lastIndex < 0 {
		return 0, nil
	}

	// Try to calculate the indicator value
	// Techan indicators return valid values even with fewer bars than the period
	// (e.g., EMA can calculate with just 1 bar, RSI needs period+1)
	value := t.indicator.Calculate(lastIndex)
	
	// Check if the value is valid (not zero or NaN)
	// For most Techan indicators, if Calculate returns a value, it's valid
	valueFloat := value.Float()
	
	// Mark as ready if we have at least 1 bar and the value is not NaN
	// Some indicators (like EMA) can work with 1 bar, others need more
	if lastIndex >= 0 && !isNaN(valueFloat) {
		t.ready = true
		return valueFloat, nil
	}

	return 0, nil
}

func (t *TechanCalculator) Value() (float64, error) {
	if !t.ready {
		return 0, fmt.Errorf("indicator not ready: need at least %d bars", t.period)
	}
	lastIndex := t.series.LastIndex()
	value := t.indicator.Calculate(lastIndex)
	return value.Float(), nil
}

func (t *TechanCalculator) Reset() {
	t.series = techan.NewTimeSeries()
	t.ready = false
}

func (t *TechanCalculator) IsReady() bool {
	return t.ready
}

// WindowSize returns the number of bars required for this indicator
func (t *TechanCalculator) WindowSize() int {
	return t.period
}

// BarsProcessed returns the number of bars processed so far
func (t *TechanCalculator) BarsProcessed() int {
	return t.series.LastIndex() + 1
}

// isNaN checks if a float64 is NaN
func isNaN(f float64) bool {
	return f != f
}
