package indicator

import (
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
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
func NewTechanCalculator(
	name string,
	indicator techan.Indicator,
	period int,
) *TechanCalculator {
	return &TechanCalculator{
		name:      name,
		series:    techan.NewTimeSeries(),
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

	candle.OpenPrice = techan.NewDecimal(bar.Open)
	candle.HighPrice = techan.NewDecimal(bar.High)
	candle.LowPrice = techan.NewDecimal(bar.Low)
	candle.ClosePrice = techan.NewDecimal(bar.Close)
	candle.Volume = techan.NewDecimal(float64(bar.Volume))

	t.series.AddCandle(candle)

	// Check if we have enough data
	lastIndex := t.series.LastIndex()
	if lastIndex >= t.period-1 {
		t.ready = true
		value := t.indicator.Calculate(lastIndex)
		return value.Float(), nil
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

