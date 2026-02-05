// Package postgres provides PostgreSQL storage backend for usage statistics.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

// Pool manages the PostgreSQL connection pool.
type Pool struct {
	pool *pgxpool.Pool
}

// NewPool creates a new connection pool from the given configuration.
func NewPool(ctx context.Context, dsn string, maxConns, minConns int32, maxLifetime, maxIdleTime string) (*Pool, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DSN cannot be empty")
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Set connection pool configuration
	config.MaxConns = maxConns
	if config.MaxConns <= 0 {
		config.MaxConns = 10
	}
	config.MinConns = minConns
	if config.MinConns < 0 {
		config.MinConns = 2
	}
	if config.MinConns > config.MaxConns {
		config.MinConns = config.MaxConns
	}

	if maxLifetime != "" {
		if duration, err := time.ParseDuration(maxLifetime); err == nil {
			config.MaxConnLifetime = duration
		} else {
			log.Warnf("Invalid max-conn-lifetime '%s', using default 1h", maxLifetime)
			config.MaxConnLifetime = time.Hour
		}
	} else {
		config.MaxConnLifetime = time.Hour
	}

	if maxIdleTime != "" {
		if duration, err := time.ParseDuration(maxIdleTime); err == nil {
			config.MaxConnIdleTime = duration
		} else {
			log.Warnf("Invalid max-conn-idle-time '%s', using default 30m", maxIdleTime)
			config.MaxConnIdleTime = 30 * time.Minute
		}
	} else {
		config.MaxConnIdleTime = 30 * time.Minute
	}

	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Infof("PostgreSQL connection pool initialized: min_conns=%d, max_conns=%d",
		config.MinConns, config.MaxConns)

	return &Pool{
		pool: pool,
	}, nil
}

// Close gracefully closes the connection pool.
// It waits for ongoing operations to complete (up to the timeout).
func (p *Pool) Close() {
	if p == nil || p.pool == nil {
		return
	}

	log.Info("Closing PostgreSQL connection pool...")
	p.pool.Close()
	log.Info("PostgreSQL connection pool closed")
}

// Ping verifies the connection to the database is alive.
func (p *Pool) Ping(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("pool is not initialized")
	}
	return p.pool.Ping(ctx)
}

// Pool returns the underlying pgxpool.Pool for direct access.
func (p *Pool) Pool() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	return p.pool
}

// Stats returns connection pool statistics.
func (p *Pool) Stats() *pgxpool.Stat {
	if p == nil || p.pool == nil {
		return nil
	}
	return p.pool.Stat()
}
