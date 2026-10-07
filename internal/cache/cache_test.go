package cache

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheSetGet(t *testing.T) {
	c, err := NewRistrettoCache(128)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer c.Close()

	c.Set("test:key:1", "hello_world", 1*time.Minute)
	// Give Ristretto a moment to commit buffer
	time.Sleep(10 * time.Millisecond)

	val, found := c.Get("test:key:1")
	if !found {
		t.Fatalf("expected key to be found")
	}
	if val.(string) != "hello_world" {
		t.Fatalf("expected 'hello_world', got %v", val)
	}
}

func TestSingleFlightStampedeProtection(t *testing.T) {
	c, err := NewRistrettoCache(128)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer c.Close()

	var computeCount int32
	var wg sync.WaitGroup
	concurrentClients := 50

	for i := 0; i < concurrentClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := c.GetOrCompute("stampede:key", 2*time.Minute, func() (interface{}, error) {
				// Simulate heavy database read
				time.Sleep(20 * time.Millisecond)
				atomic.AddInt32(&computeCount, 1)
				return "computed_data", nil
			})
			if err != nil || val != "computed_data" {
				t.Errorf("unexpected compute result: %v, err: %v", val, err)
			}
		}()
	}

	wg.Wait()

	// SingleFlight should guarantee the compute function was called exactly once
	if atomic.LoadInt32(&computeCount) != 1 {
		t.Fatalf("expected computeCount to be 1, got %d", computeCount)
	}
}

func TestCacheInvalidatePrefix(t *testing.T) {
	c, err := NewRistrettoCache(128)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer c.Close()

	c.Set("student:101:profile", "profile_101", 5*time.Minute)
	c.Set("student:101:marks", "marks_101", 5*time.Minute)
	c.Set("student:202:profile", "profile_202", 5*time.Minute)
	time.Sleep(10 * time.Millisecond)

	c.InvalidatePrefix("student:101")
	time.Sleep(10 * time.Millisecond)

	if _, found := c.Get("student:101:profile"); found {
		t.Errorf("expected student:101:profile to be invalidated")
	}
	if _, found := c.Get("student:101:marks"); found {
		t.Errorf("expected student:101:marks to be invalidated")
	}
	if _, found := c.Get("student:202:profile"); !found {
		t.Errorf("expected student:202:profile to still exist")
	}
}
