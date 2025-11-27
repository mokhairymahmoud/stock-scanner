package indicator

import (
	"fmt"
	"time"

	indicatorpkg "github.com/mohamedkhairy/stock-scanner/pkg/indicator"
)

// RegisterAllIndicators registers all available indicators (Techan + Custom)
func RegisterAllIndicators(registry *IndicatorRegistry) error {
	// Register Techan indicators
	if err := registerTechanIndicators(registry); err != nil {
		return err
	}

	// Register custom indicators (not in Techan)
	if err := registerCustomIndicators(registry); err != nil {
		return err
	}

	return nil
}

// registerTechanIndicators registers all Techan indicators
func registerTechanIndicators(registry *IndicatorRegistry) error {
	// RSI indicators
	rsiPeriods := []int{9, 14, 21}
	for _, period := range rsiPeriods {
		name := fmt.Sprintf("rsi_%d", period)
		if err := registry.Register(name,
			indicatorpkg.CreateTechanRSI(period),
			IndicatorMetadata{
				Name:        name,
				Type:        "techan",
				Description: fmt.Sprintf("Relative Strength Index (%d period)", period),
				Category:    "momentum",
				Parameters:  map[string]interface{}{"period": period},
			},
		); err != nil {
			return err
		}
	}

	// EMA indicators
	emaPeriods := []int{9, 12, 20, 21, 26, 50, 200}
	for _, period := range emaPeriods {
		name := fmt.Sprintf("ema_%d", period)
		if err := registry.Register(name,
			indicatorpkg.CreateTechanEMA(period),
			IndicatorMetadata{
				Name:        name,
				Type:        "techan",
				Description: fmt.Sprintf("Exponential Moving Average (%d period)", period),
				Category:    "trend",
				Parameters:  map[string]interface{}{"period": period},
			},
		); err != nil {
			return err
		}
	}

	// SMA indicators
	smaPeriods := []int{10, 20, 50, 200}
	for _, period := range smaPeriods {
		name := fmt.Sprintf("sma_%d", period)
		if err := registry.Register(name,
			indicatorpkg.CreateTechanSMA(period),
			IndicatorMetadata{
				Name:        name,
				Type:        "techan",
				Description: fmt.Sprintf("Simple Moving Average (%d period)", period),
				Category:    "trend",
				Parameters:  map[string]interface{}{"period": period},
			},
		); err != nil {
			return err
		}
	}

	// MACD
	if err := registry.Register("macd_12_26_9",
		indicatorpkg.CreateTechanMACD(12, 26, 9),
		IndicatorMetadata{
			Name:        "macd_12_26_9",
			Type:        "techan",
			Description: "MACD (12, 26, 9)",
			Category:    "trend",
			Parameters: map[string]interface{}{
				"fast_period":   12,
				"slow_period":   26,
				"signal_period": 9,
			},
		},
	); err != nil {
		return err
	}

	// ATR
	atrPeriods := []int{14}
	for _, period := range atrPeriods {
		name := fmt.Sprintf("atr_%d", period)
		if err := registry.Register(name,
			indicatorpkg.CreateTechanATR(period),
			IndicatorMetadata{
				Name:        name,
				Type:        "techan",
				Description: fmt.Sprintf("Average True Range (%d period)", period),
				Category:    "volatility",
				Parameters:  map[string]interface{}{"period": period},
			},
		); err != nil {
			return err
		}
	}

	// Bollinger Bands
	if err := registry.Register("bb_20_2.0",
		indicatorpkg.CreateTechanBollingerBands(20, 2.0),
		IndicatorMetadata{
			Name:        "bb_20_2.0",
			Type:        "techan",
			Description: "Bollinger Bands (20 period, 2.0 std dev)",
			Category:    "volatility",
			Parameters: map[string]interface{}{
				"period":    20,
				"multiplier": 2.0,
			},
		},
	); err != nil {
		return err
	}

	// Stochastic
	if err := registry.Register("stoch_14_3_3",
		indicatorpkg.CreateTechanStochastic(14, 3, 3),
		IndicatorMetadata{
			Name:        "stoch_14_3_3",
			Type:        "techan",
			Description: "Stochastic Oscillator (14, 3, 3)",
			Category:    "momentum",
			Parameters: map[string]interface{}{
				"k_period": 14,
				"d_period": 3,
				"smooth_k": 3,
			},
		},
	); err != nil {
		return err
	}

	return nil
}

// registerCustomIndicators registers custom indicators (not in Techan)
func registerCustomIndicators(registry *IndicatorRegistry) error {
	// VWAP indicators
	vwapWindows := []time.Duration{
		5 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
	}
	for _, window := range vwapWindows {
		name := fmt.Sprintf("vwap_%s", formatDuration(window))
		window := window // Capture loop variable
		if err := registry.Register(name,
			func() (indicatorpkg.Calculator, error) {
				return indicatorpkg.NewVWAP(window)
			},
			IndicatorMetadata{
				Name:        name,
				Type:        "custom",
				Description: fmt.Sprintf("Volume Weighted Average Price (%s window)", window),
				Category:    "price",
				Parameters:  map[string]interface{}{"window": window.String()},
			},
		); err != nil {
			return err
		}
	}

	// Volume average indicators
	volumeWindows := []time.Duration{
		5 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
	}
	for _, window := range volumeWindows {
		name := fmt.Sprintf("volume_avg_%s", formatDuration(window))
		window := window
		if err := registry.Register(name,
			func() (indicatorpkg.Calculator, error) {
				return indicatorpkg.NewVolumeAverage(window)
			},
			IndicatorMetadata{
				Name:        name,
				Type:        "custom",
				Description: fmt.Sprintf("Average Volume (%s window)", window),
				Category:    "volume",
				Parameters:  map[string]interface{}{"window": window.String()},
			},
		); err != nil {
			return err
		}
	}

	// Price change indicators
	priceChangeWindows := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
	}
	for _, window := range priceChangeWindows {
		name := fmt.Sprintf("price_change_%s_pct", formatDuration(window))
		window := window
		if err := registry.Register(name,
			func() (indicatorpkg.Calculator, error) {
				return indicatorpkg.NewPriceChange(window)
			},
			IndicatorMetadata{
				Name:        name,
				Type:        "custom",
				Description: fmt.Sprintf("Price Change Percentage (%s window)", window),
				Category:    "price",
				Parameters:  map[string]interface{}{"window": window.String()},
			},
		); err != nil {
			return err
		}
	}

	return nil
}

// formatDuration formats a duration for use in indicator names
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	return fmt.Sprintf("%dd", days)
}

