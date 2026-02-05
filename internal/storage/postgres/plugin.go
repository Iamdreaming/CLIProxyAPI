// Package postgres provides PostgreSQL storage backend for usage statistics.
package postgres

import (
	"context"
	"fmt"
	"sync"
	"time"

	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
	log "github.com/sirupsen/logrus"
)

// Plugin implements the usage.Plugin interface for PostgreSQL storage.
type Plugin struct {
	pool *Pool

	// Buffer channel for asynchronous writes
	buffer chan coreusage.Record

	// Wait group for graceful shutdown
	wg sync.WaitGroup

	// Channel to stop the worker
	stopCh chan struct{}

	// Closed flag
	closed bool
	mu     sync.RWMutex
}

// NewPlugin creates a new PostgreSQL storage plugin.
func NewPlugin(pool *Pool, bufferSize int) *Plugin {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	p := &Plugin{
		pool:   pool,
		buffer: make(chan coreusage.Record, bufferSize),
		stopCh: make(chan struct{}),
	}

	p.wg.Add(1)
	go p.worker()

	return p
}

// HandleUsage implements usage.Plugin.
// It queues the usage record for asynchronous writing to PostgreSQL.
func (p *Plugin) HandleUsage(ctx context.Context, record coreusage.Record) {
	if p == nil {
		return
	}

	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()

	if closed {
		log.Debugf("PostgreSQL plugin is closed, discarding record")
		return
	}

	select {
	case p.buffer <- record:
	default:
		log.Warn("PostgreSQL plugin buffer full, discarding record")
	}
}

// worker processes records from the buffer and writes them to PostgreSQL.
func (p *Plugin) worker() {
	defer p.wg.Done()

	for {
		select {
		case record := <-p.buffer:
			p.writeRecord(record)
		case <-p.stopCh:
			// Drain remaining records
			for len(p.buffer) > 0 {
				record := <-p.buffer
				p.writeRecord(record)
			}
			return
		}
	}
}

// writeRecord writes a single record to PostgreSQL.
func (p *Plugin) writeRecord(record coreusage.Record) {
	log.Debugf("PostgreSQL: attempting to write record - provider=%s model=%s tokens=%d failed=%v",
		record.Provider, record.Model, record.Detail.TotalTokens, record.Failed)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn := p.pool.Pool()

	const insertSQL = `
INSERT INTO usage_records (
	provider, model, api_key, auth_id, auth_index, source,
	requested_at, failed,
	input_tokens, output_tokens, reasoning_tokens, cached_tokens, total_tokens
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

	_, err := conn.Exec(ctx, insertSQL,
		record.Provider,
		record.Model,
		record.APIKey,
		record.AuthID,
		record.AuthIndex,
		record.Source,
		record.RequestedAt,
		record.Failed,
		record.Detail.InputTokens,
		record.Detail.OutputTokens,
		record.Detail.ReasoningTokens,
		record.Detail.CachedTokens,
		record.Detail.TotalTokens,
	)

	if err != nil {
		log.Errorf("Failed to write usage record to PostgreSQL: %v", err)
	}
}

// Close stops the plugin and waits for pending records to be written.
func (p *Plugin) Close() {
	if p == nil {
		return
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	close(p.stopCh)
	p.wg.Wait()
	log.Info("PostgreSQL storage plugin closed")
}

// IsActive returns true if the plugin is active (not closed).
func (p *Plugin) IsActive() bool {
	if p == nil {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return !p.closed
}

// Pool returns the underlying connection pool for query operations.
func (p *Plugin) Pool() *Pool {
	if p == nil {
		return nil
	}
	return p.pool
}

// Init initializes the PostgreSQL schema.
func Init(plugin *Plugin) error {
	if plugin == nil || plugin.pool == nil {
		return fmt.Errorf("plugin or pool is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return InitSchema(ctx, plugin.pool)
}

// InitFromConfig initializes the PostgreSQL storage backend from configuration.
// Returns the plugin if successfully initialized, or nil if disabled/failed.
func InitFromConfig(enable bool, dsn string, maxConns, minConns int32, maxLifetime, maxIdleTime string) (*Plugin, error) {
	if !enable {
		return nil, nil
	}
	if dsn == "" {
		return nil, fmt.Errorf("postgres-storage DSN is empty")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, dsn, maxConns, minConns, maxLifetime, maxIdleTime)
	if err != nil {
		return nil, fmt.Errorf("failed to create PostgreSQL pool: %w", err)
	}

	plugin := NewPlugin(pool, 1000)
	if err := Init(plugin); err != nil {
		plugin.Close()
		pool.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Info("PostgreSQL storage backend initialized successfully")
	return plugin, nil
}
