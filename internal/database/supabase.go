package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/config"
)

var globalSupabasePool *pgxpool.Pool

// InitSupabase initializes PostgreSQL connection pool for Supabase with strict bounds.
func InitSupabase(cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DATABASE_URL: %w", err)
	}

	// Strict connection limits for Supabase Transaction Pooler and 1GB VPS ceiling
	poolConfig.MaxConns = 15
	poolConfig.MinConns = 3
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgxpool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database unreachable: %w", err)
	}

	log.Println("[INFO] Successfully connected to Supabase PostgreSQL (pgxpool MaxConns=15).")
	globalSupabasePool = pool
	return pool, nil
}

// GetSupabasePool returns the active pgxpool instance.
func GetSupabasePool() *pgxpool.Pool {
	return globalSupabasePool
}
