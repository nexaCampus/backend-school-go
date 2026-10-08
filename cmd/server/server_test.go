package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nexaCampus/backend-school-go/internal/cache"
	"github.com/nexaCampus/backend-school-go/internal/database"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

func TestHealthCheck(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if body != "OK" {
		t.Fatalf("expected body 'OK', got '%s'", body)
	}
}

func TestDigitalBadgeHMAC(t *testing.T) {
	secret := "test_secret_key"
	studentID := "10211125"
	grade := "12"
	section := "E"
	rollNo := "20"
	validUntil := time.Now().Add(12 * time.Hour).Format(time.RFC3339)

	payload := fmt.Sprintf("CAMPUS-GATE|%s|%s|%s|%s|%s", studentID, grade, section, rollNo, validUntil)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	if len(sig) != 64 {
		t.Fatalf("expected 64 char hex signature, got %d", len(sig))
	}
}

func TestRefreshTokenFlow(t *testing.T) {
	secret := "super_secret_school_jwt_key_2026_stitch"
	claims := &models.StudentClaims{
		StudentID: "10211125",
		Grade:     "12",
		Section:   "E",
		FullName:  "Rajen Shaw",
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			Subject:   "10211125",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign refresh token: %v", err)
	}

	parsedClaims := &models.StudentClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenStr, parsedClaims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil || !parsedToken.Valid {
		t.Fatalf("expected valid parsed token: %v", err)
	}
	if parsedClaims.TokenType != "refresh" {
		t.Fatalf("expected token type refresh, got %s", parsedClaims.TokenType)
	}
}

func TestLocalCacheBeforeDatabaseFallback(t *testing.T) {
	// 1. Initialize local cache
	localCache, err := cache.NewRistrettoCache(64, 10000)
	if err != nil {
		t.Fatalf("failed to initialize Ristretto cache: %v", err)
	}
	defer localCache.Close()
	cache.SetDefaultCache(localCache)

	var dbQueryCount int32

	// Simulated database query function
	simulatedDBQuery := func() (interface{}, error) {
		atomic.AddInt32(&dbQueryCount, 1)
		return map[string]interface{}{
			"student_id": "STU10211125",
			"full_name":  "Rajen Shaw",
			"grade":      "12",
			"section":    "E",
		}, nil
	}

	key := "student:profile:STU10211125"

	// 1st Call: Cache Miss -> Must hit the database
	res1, err := database.GetOrCompute(key, 5*time.Minute, simulatedDBQuery)
	if err != nil {
		t.Fatalf("unexpected error on 1st call: %v", err)
	}
	if atomic.LoadInt32(&dbQueryCount) != 1 {
		t.Fatalf("expected dbQueryCount to be 1 on cache miss, got %d", dbQueryCount)
	}
	data1, ok := res1.(map[string]interface{})
	if !ok || data1["full_name"] != "Rajen Shaw" {
		t.Fatalf("unexpected data on 1st call: %+v", res1)
	}

	// 2nd Call: Cache Hit -> Must NOT hit the database
	res2, err := database.GetOrCompute(key, 5*time.Minute, simulatedDBQuery)
	if err != nil {
		t.Fatalf("unexpected error on 2nd call: %v", err)
	}
	if atomic.LoadInt32(&dbQueryCount) != 1 {
		t.Fatalf("expected dbQueryCount to remain 1 on cache hit, got %d", dbQueryCount)
	}
	data2, ok := res2.(map[string]interface{})
	if !ok || data2["full_name"] != "Rajen Shaw" {
		t.Fatalf("unexpected data on 2nd call: %+v", res2)
	}

	// 3rd Step: Invalidate cache key (e.g., student updated avatar / profile mutation)
	database.InvalidateKey(key)

	// 4th Call: Cache Miss again after invalidation -> Must query DB
	res3, err := database.GetOrCompute(key, 5*time.Minute, simulatedDBQuery)
	if err != nil {
		t.Fatalf("unexpected error on 3rd call: %v", err)
	}
	if atomic.LoadInt32(&dbQueryCount) != 2 {
		t.Fatalf("expected dbQueryCount to increment to 2 after invalidation, got %d", dbQueryCount)
	}
	data3, ok := res3.(map[string]interface{})
	if !ok || data3["full_name"] != "Rajen Shaw" {
		t.Fatalf("unexpected data on 3rd call: %+v", res3)
	}
}

func TestLocalCachePrefixInvalidation(t *testing.T) {
	localCache, err := cache.NewRistrettoCache(64, 10000)
	if err != nil {
		t.Fatalf("failed to initialize cache: %v", err)
	}
	defer localCache.Close()
	cache.SetDefaultCache(localCache)

	// Set multiple keys under prefix
	localCache.Set("homework:list:12:E:all", []string{"hw1", "hw2"}, 5*time.Minute)
	localCache.Set("homework:list:12:E:pending", []string{"hw1"}, 5*time.Minute)
	localCache.Set("notice:list:all:10", []string{"n1"}, 5*time.Minute)

	time.Sleep(10 * time.Millisecond)

	// Invalidate all homework lists
	database.InvalidatePrefix("homework:")
	time.Sleep(10 * time.Millisecond)

	if _, found := localCache.Get("homework:list:12:E:all"); found {
		t.Errorf("expected homework:list:12:E:all to be evicted")
	}
	if _, found := localCache.Get("homework:list:12:E:pending"); found {
		t.Errorf("expected homework:list:12:E:pending to be evicted")
	}
	if _, found := localCache.Get("notice:list:all:10"); !found {
		t.Errorf("expected notice:list:all:10 to remain in cache")
	}
}

