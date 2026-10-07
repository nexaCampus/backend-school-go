package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type ipBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter implements a high-throughput, memory-bounded token bucket per client IP.
type RateLimiter struct {
	mu       sync.Mutex
	capacity float64
	rate     float64 // tokens per second
	buckets  map[string]*ipBucket
}

// NewRateLimiter creates a rate limiter allowing up to burst reqs and continuous rps.
func NewRateLimiter(rate, burst float64) *RateLimiter {
	rl := &RateLimiter{
		capacity: burst,
		rate:     rate,
		buckets:  make(map[string]*ipBucket),
	}

	// Background eviction of idle IP entries to prevent memory leaks
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for ip, b := range rl.buckets {
				if now.Sub(b.lastRefill) > 5*time.Minute {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

// Middleware creates an HTTP middleware handler.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			rl.mu.Lock()
			now := time.Now()
			b, exists := rl.buckets[ip]
			if !exists {
				b = &ipBucket{
					tokens:     rl.capacity - 1,
					lastRefill: now,
				}
				rl.buckets[ip] = b
				rl.mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			// Refill tokens
			elapsed := now.Sub(b.lastRefill).Seconds()
			b.tokens += elapsed * rl.rate
			if b.tokens > rl.capacity {
				b.tokens = rl.capacity
			}
			b.lastRefill = now

			if b.tokens < 1 {
				rl.mu.Unlock()
				http.Error(w, `{"success":false,"error":"too_many_requests"}`, http.StatusTooManyRequests)
				return
			}

			b.tokens--
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	return r.RemoteAddr
}
