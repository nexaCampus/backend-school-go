package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/config"
)

var globalPool *pgxpool.Pool

// InitDB initializes the PostgreSQL connection pool using pgxpool with Render/Supabase optimized settings.
func InitDB(cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to parse DATABASE_URL: %w", err)
	}

	// Optimized connection footprint for Supabase Transaction Pooler (Port 6543)
	// and Render's free tier (< 30MB RAM footprint).
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connectivity on startup
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database unreachable on startup: %w", err)
	}

	log.Println("[INFO] Successfully connected to PostgreSQL via pgxpool.")

	// Auto-migrate tables and seed sample data
	if err := InitSchema(context.Background(), pool); err != nil {
		log.Printf("[WARN] Error ensuring schema initialization: %v", err)
	}

	globalPool = pool
	return pool, nil
}

// GetPool returns the global database pool.
func GetPool() *pgxpool.Pool {
	return globalPool
}

// GetOrCompute checks the local memory cache before falling back to a database query.
// On cache hit, it returns the cached result without querying the database.
// On cache miss, it computes the value using computeFn, caches it for ttl duration, and returns it.
func GetOrCompute(key string, ttl time.Duration, computeFn func() (interface{}, error)) (interface{}, error) {
	c := cache.GetDefaultCache()
	if c == nil {
		return computeFn()
	}
	return c.GetOrCompute(key, ttl, computeFn)
}

// InvalidatePrefix evicts all cached database items matching the specified key prefix.
func InvalidatePrefix(prefix string) {
	if c := cache.GetDefaultCache(); c != nil {
		c.InvalidatePrefix(prefix)
	}
}

// InvalidateKey evicts a single cached database item.
func InvalidateKey(key string) {
	if c := cache.GetDefaultCache(); c != nil {
		c.Delete(key)
	}
}

// WaitCache blocks until all buffered cache writes are committed.
func WaitCache() {
	if c := cache.GetDefaultCache(); c != nil {
		c.Wait()
	}
}

