package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

func TestAuthMiddleware(t *testing.T) {
	secret := "test_secret_school_2026"

	// 1. Missing Authorization header -> 401
	handler := AuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/v1/students/10211125", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for missing header, got %d", rr.Code)
	}

	// 2. Valid token -> 200 and claims in context
	claims := &models.StudentClaims{
		StudentID: "10211125",
		Grade:     "12",
		Section:   "E",
		FullName:  "RAJEN SHAW",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Subject:   "10211125",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	var capturedStudentID string
	verifyHandler := AuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cl, ok := GetClaims(r.Context())
		if ok && cl != nil {
			capturedStudentID = cl.StudentID
		}
		w.WriteHeader(http.StatusOK)
	}))

	req = httptest.NewRequest("GET", "/v1/students/10211125", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr = httptest.NewRecorder()
	verifyHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid token, got %d", rr.Code)
	}
	if capturedStudentID != "10211125" {
		t.Fatalf("expected claims student_id 10211125, got %s", capturedStudentID)
	}

	// 3. OptionalAuthMiddleware allows unauthenticated request
	optHandler := OptionalAuthMiddleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req = httptest.NewRequest("GET", "/v1/timetable?grade=12&section=E", nil)
	rr = httptest.NewRecorder()
	optHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for unauthenticated optional request, got %d", rr.Code)
	}
}
