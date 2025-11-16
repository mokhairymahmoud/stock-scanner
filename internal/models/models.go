package models

import (
	"time"
)

// Tick represents a single market data tick
type Tick struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Size      int64     `json:"size"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "trade" or "quote"
	Bid       float64   `json:"bid,omitempty"`
	Ask       float64   `json:"ask,omitempty"`
}

// Validate validates a Tick
func (t *Tick) Validate() error {
	if t.Symbol == "" {
		return ErrInvalidSymbol
	}
	if t.Price <= 0 {
		return ErrInvalidPrice
	}
	if t.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	return nil
}

// Bar1m represents a finalized 1-minute bar
type Bar1m struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    int64     `json:"volume"`
	VWAP      float64   `json:"vwap"`
}

// Validate validates a Bar1m
func (b *Bar1m) Validate() error {
	if b.Symbol == "" {
		return ErrInvalidSymbol
	}
	if b.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	if b.High < b.Low {
		return ErrInvalidBar
	}
	if b.Volume < 0 {
		return ErrInvalidVolume
	}
	return nil
}

// LiveBar represents a bar that is currently being built (not yet finalized)
type LiveBar struct {
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"` // Start of the minute
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    int64     `json:"volume"`
	VWAPNum   float64   `json:"vwap_num"`   // Numerator for VWAP calculation
	VWAPDenom float64   `json:"vwap_denom"` // Denominator for VWAP calculation
}

// ToBar1m converts a LiveBar to a finalized Bar1m
func (lb *LiveBar) ToBar1m() *Bar1m {
	vwap := 0.0
	if lb.VWAPDenom > 0 {
		vwap = lb.VWAPNum / lb.VWAPDenom
	}
	return &Bar1m{
		Symbol:    lb.Symbol,
		Timestamp: lb.Timestamp,
		Open:      lb.Open,
		High:      lb.High,
		Low:       lb.Low,
		Close:     lb.Close,
		Volume:    lb.Volume,
		VWAP:      vwap,
	}
}

// Update updates the live bar with a new tick
func (lb *LiveBar) Update(tick *Tick) {
	if lb.Open == 0 {
		lb.Open = tick.Price
		lb.High = tick.Price
		lb.Low = tick.Price
	}
	if tick.Price > lb.High {
		lb.High = tick.Price
	}
	if tick.Price < lb.Low {
		lb.Low = tick.Price
	}
	lb.Close = tick.Price
	lb.Volume += tick.Size
	lb.VWAPNum += tick.Price * float64(tick.Size)
	lb.VWAPDenom += float64(tick.Size)
}

// SymbolState represents the current state of a symbol for scanning
type SymbolState struct {
	Symbol       string                 `json:"symbol"`
	LiveBar      *LiveBar               `json:"live_bar"`
	LastFinalBars []*Bar1m              `json:"last_final_bars"` // Ring buffer of recent finalized bars
	Indicators   map[string]interface{} `json:"indicators"`       // Indicator values
	LastUpdate   time.Time              `json:"last_update"`
}

// Indicator represents a computed technical indicator
type Indicator struct {
	Symbol    string                 `json:"symbol"`
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"` // e.g., {"rsi_14": 65.5, "ema_20": 150.2}
}

// Rule represents a trading rule definition
type Rule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Conditions  []Condition `json:"conditions"`
	Cooldown    int         `json:"cooldown"` // Cooldown in seconds
	Enabled     bool        `json:"enabled"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Condition represents a single condition in a rule
type Condition struct {
	Metric   string      `json:"metric"`   // e.g., "rsi_14", "price_change_5m_pct"
	Operator string      `json:"operator"` // ">", "<", ">=", "<=", "==", "!="
	Value    interface{} `json:"value"`     // Comparison value
}

// Validate validates a Rule
func (r *Rule) Validate() error {
	if r.ID == "" {
		return ErrInvalidRuleID
	}
	if r.Name == "" {
		return ErrInvalidRuleName
	}
	if len(r.Conditions) == 0 {
		return ErrNoConditions
	}
	for _, cond := range r.Conditions {
		if err := cond.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Validate validates a Condition
func (c *Condition) Validate() error {
	if c.Metric == "" {
		return ErrInvalidMetric
	}
	validOps := map[string]bool{
		">": true, "<": true, ">=": true, "<=": true, "==": true, "!=": true,
	}
	if !validOps[c.Operator] {
		return ErrInvalidOperator
	}
	return nil
}

// Alert represents a generated alert
type Alert struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Symbol    string    `json:"symbol"`
	Timestamp time.Time `json:"timestamp"`
	Price     float64   `json:"price"`
	Message   string    `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
}

// Validate validates an Alert
func (a *Alert) Validate() error {
	if a.ID == "" {
		return ErrInvalidAlertID
	}
	if a.RuleID == "" {
		return ErrInvalidRuleID
	}
	if a.Symbol == "" {
		return ErrInvalidSymbol
	}
	if a.Timestamp.IsZero() {
		return ErrInvalidTimestamp
	}
	return nil
}

