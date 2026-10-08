package middleware

import (
	"net"
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

// NewStrictRateLimiter creates a strict rate limiter suited for brute-force prevention
// on login, authentication, and payment verification endpoints (e.g. max 5 reqs per minute).
func NewStrictRateLimiter(requestsPerMinute, burst float64) *RateLimiter {
	ratePerSec := requestsPerMinute / 60.0
	return NewRateLimiter(ratePerSec, burst)
}

// Middleware creates an HTTP middleware handler.
func (rl *RateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetClientIP(r)

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
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"success":false,"error":"too_many_requests_rate_limit_exceeded"}`, http.StatusTooManyRequests)
				return
			}

			b.tokens--
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

// GetClientIP extracts a clean client IP without port number.
func GetClientIP(r *http.Request) string {
	// 1. If X-Real-IP is present and valid, prioritize it
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		if host, _, err := net.SplitHostPort(xrip); err == nil {
			return host
		}
		if net.ParseIP(xrip) != nil {
			return xrip
		}
	}

	// 2. Check X-Forwarded-For right-most or first valid IP
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			candidate := strings.TrimSpace(parts[i])
			if host, _, err := net.SplitHostPort(candidate); err == nil {
				return host
			}
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}

	// 3. Fallback to RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
