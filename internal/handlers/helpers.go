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

// resolveStudentID retrieves student_id from query param, or falls back to JWT claims.
func resolveStudentID(r *http.Request) string {
	studentID := strings.TrimSpace(r.URL.Query().Get("student_id"))
	if studentID != "" {
		return studentID
	}
	if claims, ok := middleware.GetClaims(r.Context()); ok && claims.StudentID != "" {
		return claims.StudentID
	}
	return ""
}
