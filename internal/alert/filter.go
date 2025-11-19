package alert

import (
	"context"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// UserFilter handles user-based filtering of alerts
// For MVP, this is a simple pass-through filter
// In the future, this will filter by:
// - User subscriptions
// - Symbol watchlists
// - Rule subscriptions
type UserFilter struct {
	// TODO: Add user subscription management
	// TODO: Add symbol watchlist support
	// TODO: Add rule subscription support
}

// NewUserFilter creates a new user filter
func NewUserFilter() *UserFilter {
	return &UserFilter{}
}

// FilterAlert filters an alert based on user preferences
// For MVP, all alerts pass through (no filtering)
// In the future, this will check:
// - If user is subscribed to the rule
// - If symbol is in user's watchlist
// - If user has enabled alerts for this rule type
func (f *UserFilter) FilterAlert(ctx context.Context, alert *models.Alert) (bool, error) {
	// MVP: All alerts pass through
	// TODO: Implement user-based filtering
	logger.Debug("Filtering alert (MVP: all pass through)",
		logger.String("alert_id", alert.ID),
		logger.String("rule_id", alert.RuleID),
		logger.String("symbol", alert.Symbol),
	)
	return true, nil
}

// FilterAlerts filters multiple alerts
func (f *UserFilter) FilterAlerts(ctx context.Context, alerts []*models.Alert) ([]*models.Alert, error) {
	filtered := make([]*models.Alert, 0, len(alerts))
	for _, alert := range alerts {
		pass, err := f.FilterAlert(ctx, alert)
		if err != nil {
			logger.Warn("Failed to filter alert",
				logger.ErrorField(err),
				logger.String("alert_id", alert.ID),
			)
			continue
		}
		if pass {
			filtered = append(filtered, alert)
		}
	}
	return filtered, nil
}

