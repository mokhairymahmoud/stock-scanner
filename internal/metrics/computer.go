package metrics

import (
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// SymbolStateSnapshot is the input for metric computation
// This matches the scanner's SymbolStateSnapshot structure
type SymbolStateSnapshot struct {
	Symbol        string
	LiveBar       *models.LiveBar
	LastFinalBars []*models.Bar1m
	Indicators    map[string]float64
	LastTickTime  time.Time
	LastUpdate    time.Time

	// Session tracking
	CurrentSession   string // MarketSession as string
	SessionStartTime time.Time

	// Price references
	YesterdayClose float64
	TodayOpen      float64
	TodayClose     float64

	// Session-specific volume tracking
	PremarketVolume int64
	MarketVolume    int64
	PostmarketVolume int64

	// Trade count tracking
	TradeCount       int64
	TradeCountHistory []int64

	// Candle direction tracking
	CandleDirections map[string][]bool // timeframe -> []bool
}

// MetricComputer computes a metric value from symbol state
type MetricComputer interface {
	// Name returns the metric name (e.g., "price_change_5m_pct")
	Name() string

	// Compute computes the metric value from the snapshot
	// Returns (value, ok) where ok indicates if the metric could be computed
	Compute(snapshot *SymbolStateSnapshot) (float64, bool)

	// Dependencies returns metric names this computer depends on (for ordering)
	// Empty slice means no dependencies
	Dependencies() []string
}

