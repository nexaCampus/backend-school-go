package cache

import "time"

// Cache defines the contract for memory-bounded caching with stampede protection.
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, val interface{}, ttl time.Duration)
	Delete(key string)
	InvalidatePrefix(prefix string)
	GetOrCompute(key string, ttl time.Duration, computeFn func() (interface{}, error)) (interface{}, error)
	Len() int
	Close()
}
