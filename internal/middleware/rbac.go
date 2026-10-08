package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RequireRoles restricts route access to specified roles.
func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool)
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaims(r.Context())
			if claims == nil || (!allowed[claims.Role] && claims.Role != "superadmin" && claims.Role != "admin") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"success":false,"error":"forbidden_insufficient_permissions"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireStudentAccess prevents IDOR by ensuring the actor has access to target studentID.
func RequireStudentAccess(paramKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaims(r.Context())
			targetID := chi.URLParam(r, paramKey)
			if targetID == "" {
				targetID = r.URL.Query().Get(paramKey)
			}

			if targetID != "" && claims != nil {
				if !claims.HasStudentAccess(targetID) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"success":false,"error":"forbidden_student_idor_violation"}`))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
