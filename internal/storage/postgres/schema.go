// Package postgres provides PostgreSQL storage backend for usage statistics.
package postgres

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
)

const (
	// createTableSQL is the SQL statement to create the usage_records table.
	createTableSQL = `
CREATE TABLE IF NOT EXISTS usage_records (
	id BIGSERIAL PRIMARY KEY,
	provider VARCHAR(64) NOT NULL,
	model VARCHAR(128) NOT NULL,
	api_key VARCHAR(64),
	auth_id VARCHAR(64),
	auth_index VARCHAR(32),
	source VARCHAR(128),
	requested_at TIMESTAMPTZ NOT NULL,
	failed BOOLEAN NOT NULL DEFAULT FALSE,
	vendor_error_log TEXT,
	request_url TEXT,
	input_tokens BIGINT NOT NULL DEFAULT 0,
	output_tokens BIGINT NOT NULL DEFAULT 0,
	reasoning_tokens BIGINT NOT NULL DEFAULT 0,
	cached_tokens BIGINT NOT NULL DEFAULT 0,
	total_tokens BIGINT NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

	// checkTableExistsSQL checks if the usage_records table exists.
	checkTableExistsSQL = `
SELECT EXISTS (
	SELECT FROM information_schema.tables
	WHERE table_schema = 'public'
	AND table_name = 'usage_records'
);`
)

// createIndexesSQL contains the SQL statements to create indexes.
var createIndexesSQL = []string{
	`CREATE INDEX IF NOT EXISTS idx_usage_records_requested_at ON usage_records(requested_at);`,
	`CREATE INDEX IF NOT EXISTS idx_usage_records_provider ON usage_records(provider);`,
	`CREATE INDEX IF NOT EXISTS idx_usage_records_model ON usage_records(model);`,
	`CREATE INDEX IF NOT EXISTS idx_usage_records_api_key ON usage_records(api_key);`,
}

func ensureUsageRecordsColumns(ctx context.Context, pool *Pool) error {
	if pool == nil || pool.Pool() == nil {
		return fmt.Errorf("pool is not initialized")
	}
	conn := pool.Pool()
	statements := []string{
		"ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS vendor_error_log TEXT",
		"ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS request_url TEXT",
	}
	for _, stmt := range statements {
		if _, err := conn.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

// InitSchema initializes the database schema by creating the table and indexes if they don't exist.
func InitSchema(ctx context.Context, pool *Pool) error {
	if pool == nil || pool.Pool() == nil {
		return fmt.Errorf("pool is not initialized")
	}

	conn := pool.Pool()

	// Check if table exists
	var exists bool
	err := conn.QueryRow(ctx, checkTableExistsSQL).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if exists {
		if err := ensureUsageRecordsColumns(ctx, pool); err != nil {
			return fmt.Errorf("failed to ensure usage_records columns: %w", err)
		}
		log.Info("usage_records table already exists")
		return nil
	}

	// Create table
	_, err = conn.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create usage_records table: %w", err)
	}
	log.Info("usage_records table created successfully")

	// Create indexes
	for _, indexSQL := range createIndexesSQL {
		_, err = conn.Exec(ctx, indexSQL)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}
	log.Info("usage_records indexes created successfully")

	return nil
}
