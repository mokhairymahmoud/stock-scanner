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
	var filtersJSON, columnsJSON, colorSchemeJSON sql.NullString
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
		var filters models.ToplistFilter
		if err := json.Unmarshal([]byte(filtersJSON.String), &filters); err == nil {
			config.Filters = &filters
		}
	}

	if columnsJSON.Valid && columnsJSON.String != "" {
		if err := json.Unmarshal([]byte(columnsJSON.String), &config.Columns); err != nil {
			config.Columns = []string{}
		}
	}

	if colorSchemeJSON.Valid && colorSchemeJSON.String != "" {
		var colorScheme models.ToplistColorScheme
		if err := json.Unmarshal([]byte(colorSchemeJSON.String), &colorScheme); err == nil {
			config.ColorScheme = &colorScheme
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

	return s.scanToplistConfigs(rows)
}

// GetEnabledToplists retrieves all enabled toplists for a user (or all system toplists if userID is empty)
func (s *DatabaseToplistStore) GetEnabledToplists(ctx context.Context, userID string) ([]*models.ToplistConfig, error) {
	var query string
	var args []interface{}

	if userID == "" {
		// Get all enabled system toplists (user_id IS NULL)
		query = `
			SELECT id, user_id, name, description, metric, time_window, sort_order,
			       filters, columns, color_scheme, enabled, created_at, updated_at
			FROM toplist_configs
			WHERE user_id IS NULL AND enabled = true
			ORDER BY created_at DESC
		`
	} else {
		// Get enabled toplists for a specific user
		query = `
			SELECT id, user_id, name, description, metric, time_window, sort_order,
			       filters, columns, color_scheme, enabled, created_at, updated_at
			FROM toplist_configs
			WHERE user_id = $1 AND enabled = true
			ORDER BY created_at DESC
		`
		args = []interface{}{userID}
	}

	var rows *sql.Rows
	var err error
	if len(args) > 0 {
		rows, err = s.db.QueryContext(ctx, query, args...)
	} else {
		rows, err = s.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled toplists: %w", err)
	}
	defer rows.Close()

	return s.scanToplistConfigs(rows)
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
	filtersJSON, _ := json.Marshal(config.Filters)
	columnsJSON, _ := json.Marshal(config.Columns)
	colorSchemeJSON, _ := json.Marshal(config.ColorScheme)

	query := `
		INSERT INTO toplist_configs (
			id, user_id, name, description, metric, time_window, sort_order,
			filters, columns, color_scheme, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	var userID interface{}
	if config.UserID == "" {
		userID = nil
	} else {
		userID = config.UserID
	}

	_, err := s.db.ExecContext(ctx, query,
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
		return fmt.Errorf("failed to create toplist: %w", err)
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
	filtersJSON, _ := json.Marshal(config.Filters)
	columnsJSON, _ := json.Marshal(config.Columns)
	colorSchemeJSON, _ := json.Marshal(config.ColorScheme)

	query := `
		UPDATE toplist_configs
		SET name = $2, description = $3, metric = $4, time_window = $5, sort_order = $6,
		    filters = $7, columns = $8, color_scheme = $9, enabled = $10, updated_at = $11
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

// Close closes the database connection
func (s *DatabaseToplistStore) Close() error {
	return s.db.Close()
}

// scanToplistConfigs scans rows into ToplistConfig structs
func (s *DatabaseToplistStore) scanToplistConfigs(rows *sql.Rows) ([]*models.ToplistConfig, error) {
	var configs []*models.ToplistConfig

	for rows.Next() {
		var config models.ToplistConfig
		var userID sql.NullString
		var description sql.NullString
		var filtersJSON, columnsJSON, colorSchemeJSON sql.NullString
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
			return nil, fmt.Errorf("failed to scan toplist config: %w", err)
		}

		config.UserID = userID.String
		config.Description = description.String
		config.CreatedAt = createdAt
		config.UpdatedAt = updatedAt

		// Unmarshal JSON fields
		if filtersJSON.Valid && filtersJSON.String != "" {
			var filters models.ToplistFilter
			if err := json.Unmarshal([]byte(filtersJSON.String), &filters); err == nil {
				config.Filters = &filters
			}
		}

		if columnsJSON.Valid && columnsJSON.String != "" {
			if err := json.Unmarshal([]byte(columnsJSON.String), &config.Columns); err != nil {
				config.Columns = []string{}
			}
		}

		if colorSchemeJSON.Valid && colorSchemeJSON.String != "" {
			var colorScheme models.ToplistColorScheme
			if err := json.Unmarshal([]byte(colorSchemeJSON.String), &colorScheme); err == nil {
				config.ColorScheme = &colorScheme
			}
		}

		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return configs, nil
}
