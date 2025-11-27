package indicator

import (
	"fmt"

	"github.com/sdcoffey/techan"
)

// CreateTechanRSI creates an RSI indicator using Techan
func CreateTechanRSI(period int) func() (Calculator, error) {
	return func() (Calculator, error) {
		series := techan.NewTimeSeries()
		closePrice := techan.NewClosePriceIndicator(series)
		rsi := techan.NewRelativeStrengthIndexIndicator(closePrice, period)

		return NewTechanCalculator(
			fmt.Sprintf("rsi_%d", period),
			rsi,
			period,
		), nil
	}
}

// CreateTechanEMA creates an EMA indicator using Techan
func CreateTechanEMA(period int) func() (Calculator, error) {
	return func() (Calculator, error) {
		series := techan.NewTimeSeries()
		closePrice := techan.NewClosePriceIndicator(series)
		ema := techan.NewEMAIndicator(closePrice, period)

		return NewTechanCalculator(
			fmt.Sprintf("ema_%d", period),
			ema,
			period,
		), nil
	}
}

// CreateTechanSMA creates an SMA indicator using Techan
func CreateTechanSMA(period int) func() (Calculator, error) {
	return func() (Calculator, error) {
		series := techan.NewTimeSeries()
		closePrice := techan.NewClosePriceIndicator(series)
		sma := techan.NewMMAIndicator(closePrice, period) // MMA is SMA in Techan

		return NewTechanCalculator(
			fmt.Sprintf("sma_%d", period),
			sma,
			period,
		), nil
	}
}

// CreateTechanMACD creates a MACD indicator using Techan
func CreateTechanMACD(fastPeriod, slowPeriod, signalPeriod int) func() (Calculator, error) {
	return func() (Calculator, error) {
		series := techan.NewTimeSeries()
		closePrice := techan.NewClosePriceIndicator(series)
		macd := techan.NewMACDIndicator(closePrice, fastPeriod, slowPeriod)
		// Note: Signal line is separate in Techan, we'll use MACD line for now

		// MACD requires slowPeriod bars to be ready
		return NewTechanCalculator(
			fmt.Sprintf("macd_%d_%d_%d", fastPeriod, slowPeriod, signalPeriod),
			macd,
			slowPeriod,
		), nil
	}
}

// CreateTechanATR creates an ATR indicator using Techan
func CreateTechanATR(period int) func() (Calculator, error) {
	return func() (Calculator, error) {
		series := techan.NewTimeSeries()
		atr := techan.NewAverageTrueRangeIndicator(series, period)

		return NewTechanCalculator(
			fmt.Sprintf("atr_%d", period),
			atr,
			period,
		), nil
	}
}

// CreateTechanBollingerBands creates Bollinger Bands using Techan
// Returns the middle band (SMA) as the main indicator
func CreateTechanBollingerBands(period int, multiplier float64) func() (Calculator, error) {
	return func() (Calculator, error) {
		series := techan.NewTimeSeries()
		closePrice := techan.NewClosePriceIndicator(series)
		sma := techan.NewMMAIndicator(closePrice, period) // MMA is SMA in Techan
		// Note: We use SMA as the main indicator, but BB could be extended later
		_ = techan.NewBollingerUpperBandIndicator(sma, period, multiplier)

		return NewTechanCalculator(
			fmt.Sprintf("bb_%d_%.1f", period, multiplier),
			sma, // Use SMA as the main indicator
			period,
		), nil
	}
}

// CreateTechanStochastic creates a Stochastic Oscillator using Techan
func CreateTechanStochastic(kPeriod, dPeriod, smoothK int) func() (Calculator, error) {
	return func() (Calculator, error) {
		series := techan.NewTimeSeries()
		stochastic := techan.NewFastStochasticIndicator(series, kPeriod)
		// Note: Techan's FastStochastic uses kPeriod, D and smoothing are separate

		return NewTechanCalculator(
			fmt.Sprintf("stoch_%d_%d_%d", kPeriod, dPeriod, smoothK),
			stochastic,
			kPeriod,
		), nil
	}
}

