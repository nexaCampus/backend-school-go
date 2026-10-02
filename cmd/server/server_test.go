package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
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
