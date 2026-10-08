package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nexaCampus/backend-school-go/internal/middleware"
)

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
		}
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// resolveStudentID securely retrieves student_id, strictly guarding against IDOR.
// If an authenticated student attempts to pass another student's ID, it returns ""
// so the caller handler can reject the request with 403 Forbidden.
func resolveStudentID(r *http.Request) string {
	paramID := strings.TrimSpace(r.URL.Query().Get("student_id"))

	// 1. Check StudentClaims context (from /v1 portal group)
	if claims, ok := middleware.GetClaims(r.Context()); ok && claims != nil && claims.StudentID != "" {
		if paramID != "" && paramID != claims.StudentID {
			// IDOR blocked: Student token cannot query another student's data
			return ""
		}
		return claims.StudentID
	}

	// 2. Check UserClaims context (from /api/v1 engine group)
	if userClaims := middleware.GetUserClaims(r.Context()); userClaims != nil {
		if paramID != "" {
			if !userClaims.HasStudentAccess(paramID) {
				return ""
			}
			return paramID
		}
		return userClaims.UserID
	}

	return paramID
}

// VerifyStudentOwnership checks whether the caller has rights to the specified targetStudentID.
func VerifyStudentOwnership(r *http.Request, targetStudentID string) bool {
	if targetStudentID == "" {
		return false
	}
	if claims, ok := middleware.GetClaims(r.Context()); ok && claims != nil {
		return claims.StudentID == targetStudentID
	}
	if userClaims := middleware.GetUserClaims(r.Context()); userClaims != nil {
		return userClaims.HasStudentAccess(targetStudentID)
	}
	return false
}
