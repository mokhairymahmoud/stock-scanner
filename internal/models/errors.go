package models

import "errors"

var (
	ErrInvalidSymbol    = errors.New("invalid symbol")
	ErrInvalidPrice     = errors.New("invalid price")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrInvalidBar       = errors.New("invalid bar (high < low)")
	ErrInvalidVolume    = errors.New("invalid volume")
	ErrInvalidRuleID    = errors.New("invalid rule ID")
	ErrInvalidRuleName  = errors.New("invalid rule name")
	ErrNoConditions     = errors.New("rule must have at least one condition")
	ErrInvalidMetric    = errors.New("invalid metric")
	ErrInvalidOperator  = errors.New("invalid operator")
	ErrInvalidAlertID   = errors.New("invalid alert ID")
)

