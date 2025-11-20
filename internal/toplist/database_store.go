package toplist

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

// DatabaseToplistStore is a TimescaleDB-backed implementation of ToplistStore
type DatabaseToplistStore struct {
	db       *sql.DB
	dbConfig config.DatabaseConfig
}

// NewDatabaseToplistStore creates a new database-backed toplist store
func NewDatabaseToplistStore(dbConfig config.DatabaseConfig) (*DatabaseToplistStore, error) {
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

	store := &DatabaseToplistStore{
		db:       db,
		dbConfig: dbConfig,
	}

	logger.Info("Database toplist store initialized",
		logger.String("host", dbConfig.Host),
		logger.Int("port", dbConfig.Port),
		logger.String("database", dbConfig.Database),
	)

	return store, nil
}

// GetToplistConfig retrieves a toplist configuration by ID
func (s *DatabaseToplistStore) GetToplistConfig(ctx context.Context, toplistID string) (*models.ToplistConfig, error) {
	query := `
		SELECT id, user_id, name, description, metric, time_window, sort_order,
		       filters, columns, color_scheme, enabled, created_at, updated_at
		FROM toplist_configs
		WHERE id = $1
	`

	var config models.ToplistConfig
	var userID sql.NullString
	var description sql.NullString
	var filtersJSON sql.NullString
	var columnsJSON sql.NullString
	var colorSchemeJSON sql.NullString
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx, query, toplistID).Scan(
		&config.ID,
		&userID,
		&config.Name,
		&description,
		&config.Metric,
		&config.TimeWindow,
		&config.SortOrder,
		&filtersJSON,
		&columnsJSON,
		&colorSchemeJSON,
		&config.Enabled,
		&createdAt,
		&updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("toplist not found: %s", toplistID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query toplist: %w", err)
	}

	config.UserID = userID.String
	config.Description = description.String
	config.CreatedAt = createdAt
	config.UpdatedAt = updatedAt

	// Unmarshal JSON fields
	if filtersJSON.Valid && filtersJSON.String != "" {
		if err := json.Unmarshal([]byte(filtersJSON.String), &config.Filters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal filters: %w", err)
		}
	}

	if columnsJSON.Valid && columnsJSON.String != "" {
		if err := json.Unmarshal([]byte(columnsJSON.String), &config.Columns); err != nil {
			return nil, fmt.Errorf("failed to unmarshal columns: %w", err)
		}
	}

	if colorSchemeJSON.Valid && colorSchemeJSON.String != "" {
		if err := json.Unmarshal([]byte(colorSchemeJSON.String), &config.ColorScheme); err != nil {
			return nil, fmt.Errorf("failed to unmarshal color_scheme: %w", err)
		}
	}

	return &config, nil
}

// GetUserToplists retrieves all toplists for a user
func (s *DatabaseToplistStore) GetUserToplists(ctx context.Context, userID string) ([]*models.ToplistConfig, error) {
	query := `
		SELECT id, user_id, name, description, metric, time_window, sort_order,
		       filters, columns, color_scheme, enabled, created_at, updated_at
		FROM toplist_configs
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user toplists: %w", err)
	}
	defer rows.Close()

	var configs []*models.ToplistConfig
	for rows.Next() {
		config, err := s.scanToplistConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return configs, nil
}

// GetEnabledToplists retrieves all enabled toplists
func (s *DatabaseToplistStore) GetEnabledToplists(ctx context.Context) ([]*models.ToplistConfig, error) {
	query := `
		SELECT id, user_id, name, description, metric, time_window, sort_order,
		       filters, columns, color_scheme, enabled, created_at, updated_at
		FROM toplist_configs
		WHERE enabled = true
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled toplists: %w", err)
	}
	defer rows.Close()

	var configs []*models.ToplistConfig
	for rows.Next() {
		config, err := s.scanToplistConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return configs, nil
}

// CreateToplist creates a new toplist configuration
func (s *DatabaseToplistStore) CreateToplist(ctx context.Context, config *models.ToplistConfig) error {
	// Validate config
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid toplist config: %w", err)
	}

	// Set timestamps
	now := time.Now()
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	config.UpdatedAt = now

	// Marshal JSON fields
	filtersJSON, err := json.Marshal(config.Filters)
	if err != nil {
		return fmt.Errorf("failed to marshal filters: %w", err)
	}

	columnsJSON, err := json.Marshal(config.Columns)
	if err != nil {
		return fmt.Errorf("failed to marshal columns: %w", err)
	}

	colorSchemeJSON, err := json.Marshal(config.ColorScheme)
	if err != nil {
		return fmt.Errorf("failed to marshal color_scheme: %w", err)
	}

	query := `
		INSERT INTO toplist_configs (
			id, user_id, name, description, metric, time_window, sort_order,
			filters, columns, color_scheme, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	var userID interface{}
	if config.UserID != "" {
		userID = config.UserID
	}

	_, err = s.db.ExecContext(ctx, query,
		config.ID,
		userID,
		config.Name,
		config.Description,
		config.Metric,
		config.TimeWindow,
		config.SortOrder,
		string(filtersJSON),
		string(columnsJSON),
		string(colorSchemeJSON),
		config.Enabled,
		config.CreatedAt,
		config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert toplist: %w", err)
	}

	return nil
}

// UpdateToplist updates an existing toplist configuration
func (s *DatabaseToplistStore) UpdateToplist(ctx context.Context, config *models.ToplistConfig) error {
	// Validate config
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid toplist config: %w", err)
	}

	// Update timestamp
	config.UpdatedAt = time.Now()

	// Marshal JSON fields
	filtersJSON, err := json.Marshal(config.Filters)
	if err != nil {
		return fmt.Errorf("failed to marshal filters: %w", err)
	}

	columnsJSON, err := json.Marshal(config.Columns)
	if err != nil {
		return fmt.Errorf("failed to marshal columns: %w", err)
	}

	colorSchemeJSON, err := json.Marshal(config.ColorScheme)
	if err != nil {
		return fmt.Errorf("failed to marshal color_scheme: %w", err)
	}

	query := `
		UPDATE toplist_configs SET
			name = $2,
			description = $3,
			metric = $4,
			time_window = $5,
			sort_order = $6,
			filters = $7,
			columns = $8,
			color_scheme = $9,
			enabled = $10,
			updated_at = $11
		WHERE id = $1
	`

	result, err := s.db.ExecContext(ctx, query,
		config.ID,
		config.Name,
		config.Description,
		config.Metric,
		config.TimeWindow,
		config.SortOrder,
		string(filtersJSON),
		string(columnsJSON),
		string(colorSchemeJSON),
		config.Enabled,
		config.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update toplist: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("toplist not found: %s", config.ID)
	}

	return nil
}

// DeleteToplist deletes a toplist configuration
func (s *DatabaseToplistStore) DeleteToplist(ctx context.Context, toplistID string) error {
	query := `DELETE FROM toplist_configs WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, toplistID)
	if err != nil {
		return fmt.Errorf("failed to delete toplist: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("toplist not found: %s", toplistID)
	}

	return nil
}

// scanToplistConfig scans a row into a ToplistConfig
func (s *DatabaseToplistStore) scanToplistConfig(rows *sql.Rows) (*models.ToplistConfig, error) {
	var config models.ToplistConfig
	var userID sql.NullString
	var description sql.NullString
	var filtersJSON sql.NullString
	var columnsJSON sql.NullString
	var colorSchemeJSON sql.NullString
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&config.ID,
		&userID,
		&config.Name,
		&description,
		&config.Metric,
		&config.TimeWindow,
		&config.SortOrder,
		&filtersJSON,
		&columnsJSON,
		&colorSchemeJSON,
		&config.Enabled,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	config.UserID = userID.String
	config.Description = description.String
	config.CreatedAt = createdAt
	config.UpdatedAt = updatedAt

	// Unmarshal JSON fields
	if filtersJSON.Valid && filtersJSON.String != "" {
		if err := json.Unmarshal([]byte(filtersJSON.String), &config.Filters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal filters: %w", err)
		}
	}

	if columnsJSON.Valid && columnsJSON.String != "" {
		if err := json.Unmarshal([]byte(columnsJSON.String), &config.Columns); err != nil {
			return nil, fmt.Errorf("failed to unmarshal columns: %w", err)
		}
	}

	if colorSchemeJSON.Valid && colorSchemeJSON.String != "" {
		if err := json.Unmarshal([]byte(colorSchemeJSON.String), &config.ColorScheme); err != nil {
			return nil, fmt.Errorf("failed to unmarshal color_scheme: %w", err)
		}
	}

	return &config, nil
}

// Close closes the database connection
func (s *DatabaseToplistStore) Close() error {
	return s.db.Close()
}

