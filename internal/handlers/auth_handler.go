package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type AuthHandler struct {
	pool      *pgxpool.Pool
	jwtSecret string
}

func NewAuthHandler(pool *pgxpool.Pool, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		pool:      pool,
		jwtSecret: jwtSecret,
	}
}

// Verify authenticates student credentials (student_id and dob) and issues a JWT token.
// Route: POST /v1/auth/verify
func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var req models.StudentVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.StudentID = strings.TrimSpace(req.StudentID)
	req.DOB = strings.TrimSpace(req.DOB)

	if req.StudentID == "" || req.DOB == "" {
		respondError(w, http.StatusBadRequest, "student_id and dob are required")
		return
	}

	// Validate DOB format YYYY-MM-DD
	if _, err := time.Parse("2006-01-02", req.DOB); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid date of birth format, expected YYYY-MM-DD")
		return
	}

	query := `
		SELECT
			id, student_id, dob, full_name, display_name, grade, section, roll_no,
			blood_group, father_name, mother_name, contact_no, email, address,
			school_name, academic_year, avatar_url, smart_card_balance, canteen_balance,
			attendance_pct, streak_days, created_at, updated_at
		FROM students
		WHERE student_id = $1 AND dob = $2
		LIMIT 1;
	`

	var student models.Student
	err := h.pool.QueryRow(r.Context(), query, req.StudentID, req.DOB).Scan(
		&student.ID,
		&student.StudentID,
		&student.DOB,
		&student.FullName,
		&student.DisplayName,
		&student.Grade,
		&student.Section,
		&student.RollNo,
		&student.BloodGroup,
		&student.FatherName,
		&student.MotherName,
		&student.ContactNo,
		&student.Email,
		&student.Address,
		&student.SchoolName,
		&student.AcademicYear,
		&student.AvatarURL,
		&student.SmartCardBalance,
		&student.CanteenBalance,
		&student.AttendancePct,
		&student.StreakDays,
		&student.CreatedAt,
		&student.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusUnauthorized, "Invalid student ID or date of birth")
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to authenticate student")
		return
	}

	// 1. Issue 7-day expiration access token
	accessToken, err := h.generateToken(student.StudentID, student.Grade, student.Section, student.FullName, "access", 7*24*time.Hour)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to sign access token")
		return
	}

	// 2. Issue 30-day expiration refresh token
	refreshToken, err := h.generateToken(student.StudentID, student.Grade, student.Section, student.FullName, "refresh", 30*24*time.Hour)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to sign refresh token")
		return
	}

	// Set HttpOnly cookie for web client security against XSS token exfiltration
	isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     "school_auth_token",
		Value:    accessToken,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteStrictMode,
	})

	respondJSON(w, http.StatusOK, models.StudentVerifyResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		Student:      student,
	})
}

// Refresh rotates expired access tokens using a valid refresh token.
// Route: POST /v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		respondError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	claims := &models.StudentClaims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		respondError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	// Generate new access and refresh tokens
	newAccessToken, err := h.generateToken(claims.StudentID, claims.Grade, claims.Section, claims.FullName, "access", 7*24*time.Hour)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate new access token")
		return
	}

	newRefreshToken, err := h.generateToken(claims.StudentID, claims.Grade, claims.Section, claims.FullName, "refresh", 30*24*time.Hour)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate new refresh token")
		return
	}

	// Set rotated HttpOnly cookie for web client security
	isSecure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     "school_auth_token",
		Value:    newAccessToken,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteStrictMode,
	})

	respondJSON(w, http.StatusOK, models.RefreshTokenResponse{
		Token:        newAccessToken,
		RefreshToken: newRefreshToken,
	})
}

func (h *AuthHandler) generateToken(studentID, grade, section, fullName, tokenType string, duration time.Duration) (string, error) {
	claims := &models.StudentClaims{
		StudentID: studentID,
		Grade:     grade,
		Section:   section,
		FullName:  fullName,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "nexaCampus-school-portal",
			Subject:   studentID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
