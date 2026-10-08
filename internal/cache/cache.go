package cache

import (
	"sync"
	"time"
)

// Cache defines the contract for memory-bounded caching with stampede protection.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, val interface{}, ttl time.Duration)
	Delete(key string)
	InvalidatePrefix(prefix string)
	GetOrCompute(key string, ttl time.Duration, computeFn func() (interface{}, error)) (interface{}, error)
	Len() int
	Wait()
	Close()
}

var (
	defaultCache   Cache
	defaultCacheMu sync.RWMutex
)

// SetDefaultCache sets the global default cache instance.
func SetDefaultCache(c Cache) {
	defaultCacheMu.Lock()
	defer defaultCacheMu.Unlock()
	defaultCache = c
}

// GetDefaultCache returns the global default cache, lazily initializing a 64MB instance if not yet set.
func GetDefaultCache() Cache {
	defaultCacheMu.RLock()
	if defaultCache != nil {
		defer defaultCacheMu.RUnlock()
		return defaultCache
	}
	defaultCacheMu.RUnlock()

	defaultCacheMu.Lock()
	defer defaultCacheMu.Unlock()
	if defaultCache == nil {
		c, err := NewRistrettoCache(64)
		if err == nil {
			defaultCache = c
		}
	}
	return defaultCache
}

