package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis configuration
type Config struct {
	URL      string
	Password string
	DB       int
}

// NewClient creates a new Redis client
func NewClient(ctx context.Context, cfg *Config) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if cfg.Password != "" {
		opt.Password = cfg.Password
	}
	if cfg.DB != 0 {
		opt.DB = cfg.DB
	}

	client := redis.NewClient(opt)

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}
