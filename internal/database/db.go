package database

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
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

	cleanedURL := CleanDatabaseURL(cfg.DatabaseURL)
	poolConfig, err := pgxpool.ParseConfig(cleanedURL)
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

// CleanDatabaseURL sanitizes common formatting errors in connection strings,
// such as unescaped special characters in passwords or accidental square brackets.
func CleanDatabaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "postgres://") && !strings.HasPrefix(raw, "postgresql://") {
		return raw
	}

	if _, err := pgxpool.ParseConfig(raw); err == nil {
		return raw
	}

	schemeEnd := strings.Index(raw, "://")
	if schemeEnd == -1 {
		return raw
	}
	scheme := raw[:schemeEnd]
	afterScheme := raw[schemeEnd+3:]

	lastAt := strings.LastIndex(afterScheme, "@")
	if lastAt == -1 {
		return raw
	}

	userInfo := afterScheme[:lastAt]
	hostAndRest := afterScheme[lastAt+1:]

	colonIdx := strings.Index(userInfo, ":")
	if colonIdx == -1 {
		return raw
	}

	user := userInfo[:colonIdx]
	pass := userInfo[colonIdx+1:]

	// Strip literal brackets if user accidentally kept [YOUR_PASSWORD]
	pass = strings.TrimPrefix(pass, "[")
	pass = strings.TrimSuffix(pass, "]")

	// Percent-encode password for URL compliance
	escapedPass := url.QueryEscape(pass)

	fixed := fmt.Sprintf("%s://%s:%s@%s", scheme, user, escapedPass, hostAndRest)
	if _, err := pgxpool.ParseConfig(fixed); err == nil {
		return fixed
	}

	return raw
}


