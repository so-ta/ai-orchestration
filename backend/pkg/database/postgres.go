package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database configuration
type Config struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// DefaultConfig returns default database configuration
func DefaultConfig(url string) *Config {
	return &Config{
		URL:             url,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
	}
}

// NewPool creates a new PostgreSQL connection pool
func NewPool(ctx context.Context, cfg *Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// SetTenantContext sets the tenant ID in the database session
// The tenantID must be a valid UUID to prevent SQL injection
func SetTenantContext(ctx context.Context, pool *pgxpool.Pool, tenantID string) error {
	// Validate that tenantID is a valid UUID to prevent SQL injection
	parsedID, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant ID format: %w", err)
	}
	_, err = pool.Exec(ctx, fmt.Sprintf("SET app.current_tenant = '%s'", parsedID.String()))
	return err
}
