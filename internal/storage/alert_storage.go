package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// TimescaleAlertStorage implements AlertStorage interface for TimescaleDB
type TimescaleAlertStorage struct {
	db       *sql.DB
	dbConfig config.DatabaseConfig
}

// NewTimescaleAlertStorage creates a new TimescaleDB alert storage
func NewTimescaleAlertStorage(dbConfig config.DatabaseConfig) (*TimescaleAlertStorage, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Database,
		dbConfig.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(dbConfig.MaxConnections)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &TimescaleAlertStorage{
		db:       db,
		dbConfig: dbConfig,
	}

	logger.Info("TimescaleDB alert storage initialized",
		logger.String("host", dbConfig.Host),
		logger.Int("port", dbConfig.Port),
		logger.String("database", dbConfig.Database),
	)

	return storage, nil
}

// WriteAlert writes a single alert (not used in production, but required by interface)
func (s *TimescaleAlertStorage) WriteAlert(ctx context.Context, alert *models.Alert) error {
	return s.WriteAlerts(ctx, []*models.Alert{alert})
}

// WriteAlerts writes multiple alerts (not used in production, alerts are written by alert service)
func (s *TimescaleAlertStorage) WriteAlerts(ctx context.Context, alerts []*models.Alert) error {
	// This is a read-only interface for the API
	// Alerts are written by the alert service
	return fmt.Errorf("write operations not supported via this interface")
}

// GetAlerts retrieves alerts with filtering options
func (s *TimescaleAlertStorage) GetAlerts(ctx context.Context, filter AlertFilter) ([]*models.Alert, error) {
	query := `
		SELECT id, rule_id, rule_name, symbol, timestamp, price, message, metadata, trace_id
		FROM alert_history
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.Symbol != "" {
		query += fmt.Sprintf(" AND symbol = $%d", argIndex)
		args = append(args, filter.Symbol)
		argIndex++
	}

	if filter.RuleID != "" {
		query += fmt.Sprintf(" AND rule_id = $%d", argIndex)
		args = append(args, filter.RuleID)
		argIndex++
	}

	if !filter.StartTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, filter.StartTime)
		argIndex++
	}

	if !filter.EndTime.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, filter.EndTime)
		argIndex++
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
		argIndex++
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		var alert models.Alert
		var metadataJSON sql.NullString

		if err := rows.Scan(
			&alert.ID,
			&alert.RuleID,
			&alert.RuleName,
			&alert.Symbol,
			&alert.Timestamp,
			&alert.Price,
			&alert.Message,
			&metadataJSON,
			&alert.TraceID,
		); err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		// Unmarshal metadata if present
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &alert.Metadata); err != nil {
				logger.Warn("Failed to unmarshal alert metadata",
					logger.ErrorField(err),
					logger.String("alert_id", alert.ID),
				)
			}
		}

		alerts = append(alerts, &alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return alerts, nil
}

// GetAlert retrieves a single alert by ID
func (s *TimescaleAlertStorage) GetAlert(ctx context.Context, alertID string) (*models.Alert, error) {
	query := `
		SELECT id, rule_id, rule_name, symbol, timestamp, price, message, metadata, trace_id
		FROM alert_history
		WHERE id = $1
	`

	var alert models.Alert
	var metadataJSON sql.NullString

	err := s.db.QueryRowContext(ctx, query, alertID).Scan(
		&alert.ID,
		&alert.RuleID,
		&alert.RuleName,
		&alert.Symbol,
		&alert.Timestamp,
		&alert.Price,
		&alert.Message,
		&metadataJSON,
		&alert.TraceID,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query alert: %w", err)
	}

	// Unmarshal metadata if present
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &alert.Metadata); err != nil {
			logger.Warn("Failed to unmarshal alert metadata",
				logger.ErrorField(err),
				logger.String("alert_id", alert.ID),
			)
		}
	}

	return &alert, nil
}

// Close closes the database connection
func (s *TimescaleAlertStorage) Close() error {
	return s.db.Close()
}

