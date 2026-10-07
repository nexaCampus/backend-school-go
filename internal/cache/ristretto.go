package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"golang.org/x/sync/singleflight"
)

// RistrettoCache implements memory-bounded, TinyLFU-evicted caching with SingleFlight stampede guard.
type RistrettoCache struct {
	cache        *ristretto.Cache
	requestGroup singleflight.Group
	keyTracker   sync.Map // string -> int64 (expiration timestamp)
}

// NewRistrettoCache initializes a bounded cache instance.
func NewRistrettoCache(maxCostMB int64, numCounters ...int64) (*RistrettoCache, error) {
	if maxCostMB <= 0 {
		maxCostMB = 128
	}

	counters := int64(1e7)
	if len(numCounters) > 0 && numCounters[0] > 0 {
		counters = numCounters[0]
	}

	c, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: counters,               // keys tracked for tinyLFU frequency
		MaxCost:     maxCostMB * 1024 * 1024, // Strict memory ceiling (128 MB)
		BufferItems: 64,                      // 64-item ring buffer per counter strip
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ristretto cache: %w", err)
	}

	rc := &RistrettoCache{
		cache: c,
	}

	// Background cleaner for stale key tracker entries every 5 minutes
	go rc.cleanExpiredKeys()

	return rc, nil
}

// Get retrieves an item from cache if present.
func (c *RistrettoCache) Get(key string) (interface{}, bool) {
	return c.cache.Get(key)
}

// Set stores an item with an adaptive TTL.
func (c *RistrettoCache) Set(key string, val interface{}, ttl time.Duration) {
	if ttl <= 0 || val == nil {
		return
	}

	c.keyTracker.Store(key, time.Now().Add(ttl).Unix())
	c.cache.SetWithTTL(key, val, 1, ttl)
}

// Delete evicts a specific key from cache.
func (c *RistrettoCache) Delete(key string) {
	c.cache.Del(key)
	c.keyTracker.Delete(key)
}

// InvalidatePrefix evicts all keys starting with the specified prefix.
func (c *RistrettoCache) InvalidatePrefix(prefix string) {
	c.keyTracker.Range(func(k, _ interface{}) bool {
		keyStr, ok := k.(string)
		if ok && strings.HasPrefix(keyStr, prefix) {
			c.cache.Del(keyStr)
			c.keyTracker.Delete(keyStr)
		}
		return true
	})
}

// GetOrCompute uses SingleFlight to deduplicate concurrent cache misses and compute the value once.
func (c *RistrettoCache) GetOrCompute(key string, ttl time.Duration, computeFn func() (interface{}, error)) (interface{}, error) {
	if ttl <= 0 {
		return computeFn()
	}

	if val, found := c.cache.Get(key); found && val != nil {
		return val, nil
	}

	// SingleFlight ensures only 1 DB query executes among concurrent identical callers
	v, err, _ := c.requestGroup.Do(key, func() (interface{}, error) {
		// Double-check cache inside singleflight
		if val, found := c.cache.Get(key); found && val != nil {
			return val, nil
		}

		data, err := computeFn()
		if err != nil {
			return nil, err
		}

		if data != nil {
			c.Set(key, data, ttl)
		}
		return data, nil
	})

	if err != nil {
		return nil, err
	}
	return v, nil
}

// Len returns approximate count of active tracked keys.
func (c *RistrettoCache) Len() int {
	count := 0
	c.keyTracker.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}

// Close releases cache resources.
func (c *RistrettoCache) Close() {
	if c.cache != nil {
		c.cache.Close()
	}
}

func (c *RistrettoCache) cleanExpiredKeys() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		nowUnix := time.Now().Unix()
		c.keyTracker.Range(func(k, v interface{}) bool {
			exp, ok := v.(int64)
			if ok && exp < nowUnix {
				c.keyTracker.Delete(k)
			}
			return true
		})
	}
}
