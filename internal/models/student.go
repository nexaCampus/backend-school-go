package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Student represents a student profile record in the school portal.
type Student struct {
	ID               string    `json:"id"`
	StudentID        string    `json:"student_id"`
	DOB              string    `json:"dob"`
	FullName         string    `json:"name"`
	DisplayName      string    `json:"display_name"`
	Grade            string    `json:"grade"`
	Section          string    `json:"section"`
	RollNo           string    `json:"roll_no"`
	BloodGroup       string    `json:"blood_group"`
	FatherName       string    `json:"father_name"`
	MotherName       string    `json:"mother_name"`
	ContactNo        string    `json:"contact_no"`
	Email            string    `json:"email"`
	Address          string    `json:"address"`
	SchoolName       string    `json:"school_name"`
	AcademicYear     string    `json:"academic_year"`
	AvatarURL        *string   `json:"avatar_url,omitempty"`
	SmartCardBalance float64   `json:"smart_card_balance"`
	CanteenBalance   float64   `json:"canteen_balance"`
	AttendancePct    int       `json:"attendance_pct"`
	StreakDays       int       `json:"streak_days"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// StudentVerifyRequest is the payload for POST /v1/auth/verify
type StudentVerifyRequest struct {
	StudentID string `json:"student_id"`
	DOB       string `json:"dob"`
}

// StudentVerifyResponse is the response for POST /v1/auth/verify
type StudentVerifyResponse struct {
	Token        string  `json:"token"`
	RefreshToken string  `json:"refresh_token"`
	Student      Student `json:"student"`
}

// RefreshTokenRequest is the payload for POST /v1/auth/refresh
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshTokenResponse is the payload returned by POST /v1/auth/refresh
type RefreshTokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// UpdateAvatarRequest is the payload for PUT /v1/students/{id}/avatar
type UpdateAvatarRequest struct {
	AvatarURL string `json:"avatar_url"`
}

// DigitalBadgeResponse represents the cryptographic QR pass returned by GET /v1/students/{id}/digital-badge
type DigitalBadgeResponse struct {
	StudentID  string `json:"student_id"`
	Name       string `json:"name"`
	Grade      string `json:"grade"`
	Section    string `json:"section"`
	RollNo     string `json:"roll_no"`
	SchoolName string `json:"school_name"`
	QRCodeData string `json:"qr_code"`
	ValidUntil string `json:"valid_until"`
	Signature  string `json:"signature"`
}

// StudentClaims encapsulates JWT claims for an authenticated student.
type StudentClaims struct {
	StudentID string `json:"student_id"`
	Grade     string `json:"grade"`
	Section   string `json:"section"`
	FullName  string `json:"full_name"`
	TokenType string `json:"token_type,omitempty"` // "access" or "refresh"
	jwt.RegisteredClaims
}
