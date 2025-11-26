package rules

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

// DatabaseRuleStore is a TimescaleDB-backed implementation of RuleStore
type DatabaseRuleStore struct {
	db       *sql.DB
	dbConfig config.DatabaseConfig
}

// NewDatabaseRuleStore creates a new database-backed rule store
func NewDatabaseRuleStore(dbConfig config.DatabaseConfig) (*DatabaseRuleStore, error) {
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

	store := &DatabaseRuleStore{
		db:       db,
		dbConfig: dbConfig,
	}

	logger.Info("Database rule store initialized",
		logger.String("host", dbConfig.Host),
		logger.Int("port", dbConfig.Port),
		logger.String("database", dbConfig.Database),
	)

	return store, nil
}

// GetRule retrieves a rule by ID
func (s *DatabaseRuleStore) GetRule(id string) (*models.Rule, error) {
	query := `
		SELECT id, name, description, conditions, enabled, created_at, updated_at, version
		FROM rules
		WHERE id = $1
	`

	var rule models.Rule
	var conditionsJSON []byte
	var createdAt, updatedAt time.Time
	var version int

	err := s.db.QueryRow(query, id).Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&conditionsJSON,
		&rule.Enabled,
		&createdAt,
		&updatedAt,
		&version,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("rule not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query rule: %w", err)
	}

	// Unmarshal conditions
	if err := json.Unmarshal(conditionsJSON, &rule.Conditions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conditions: %w", err)
	}

	rule.CreatedAt = createdAt
	rule.UpdatedAt = updatedAt

	return &rule, nil
}

// GetAllRules retrieves all rules
func (s *DatabaseRuleStore) GetAllRules() ([]*models.Rule, error) {
	query := `
		SELECT id, name, description, conditions, enabled, created_at, updated_at, version
		FROM rules
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []*models.Rule
	for rows.Next() {
		var rule models.Rule
		var conditionsJSON []byte
		var createdAt, updatedAt time.Time
		var version int

		if err := rows.Scan(
			&rule.ID,
			&rule.Name,
			&rule.Description,
			&conditionsJSON,
			&rule.Enabled,
			&createdAt,
			&updatedAt,
			&version,
		); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		// Unmarshal conditions
		if err := json.Unmarshal(conditionsJSON, &rule.Conditions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal conditions: %w", err)
		}

		rule.CreatedAt = createdAt
		rule.UpdatedAt = updatedAt

		rules = append(rules, &rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return rules, nil
}

// AddRule adds a new rule
func (s *DatabaseRuleStore) AddRule(rule *models.Rule) error {
	// Marshal conditions to JSON
	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("failed to marshal conditions: %w", err)
	}

	query := `
		INSERT INTO rules (id, name, description, conditions, enabled, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
		    description = EXCLUDED.description,
		    conditions = EXCLUDED.conditions,
		    enabled = EXCLUDED.enabled,
		    updated_at = EXCLUDED.updated_at,
		    version = rules.version + 1
	`

	_, err = s.db.Exec(query,
		rule.ID,
		rule.Name,
		rule.Description,
		conditionsJSON,
		rule.Enabled,
		rule.CreatedAt,
		rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert rule: %w", err)
	}

	return nil
}

// UpdateRule updates an existing rule
func (s *DatabaseRuleStore) UpdateRule(rule *models.Rule) error {
	// Marshal conditions to JSON
	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("failed to marshal conditions: %w", err)
	}

	query := `
		UPDATE rules
		SET name = $2,
		    description = $3,
		    conditions = $4,
		    enabled = $5,
		    updated_at = $6,
		    version = version + 1
		WHERE id = $1
	`

	result, err := s.db.Exec(query,
		rule.ID,
		rule.Name,
		rule.Description,
		conditionsJSON,
		rule.Enabled,
		rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	return nil
}

// DeleteRule deletes a rule by ID
func (s *DatabaseRuleStore) DeleteRule(id string) error {
	query := `DELETE FROM rules WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	return nil
}

// EnableRule enables a rule
func (s *DatabaseRuleStore) EnableRule(id string) error {
	query := `UPDATE rules SET enabled = true, updated_at = NOW() WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to enable rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	return nil
}

// DisableRule disables a rule
func (s *DatabaseRuleStore) DisableRule(id string) error {
	query := `UPDATE rules SET enabled = false, updated_at = NOW() WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to disable rule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	return nil
}

// Close closes the database connection
func (s *DatabaseRuleStore) Close() error {
	return s.db.Close()
}

