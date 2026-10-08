package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type userClaimsKey struct{}

// JWTAuth verifies Bearer tokens (or HttpOnly auth cookies) and attaches UserClaims to request context.
func JWTAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenStr string

			// 1. Prioritize Authorization header: 'Bearer <token>'
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			} else if cookie, err := r.Cookie("school_auth_token"); err == nil && cookie.Value != "" {
				// 2. Fall back to secure HttpOnly cookie to mitigate XSS local storage theft
				tokenStr = cookie.Value
			}

			// Reject missing token unless explicitly enabled DEV mock mode in non-production
			if tokenStr == "" {
				isDevMock := os.Getenv("DEV_MOCK_AUTH") == "true" &&
					os.Getenv("ENV") != "production" &&
					os.Getenv("RENDER") != "true"
				if isDevMock {
					defaultClaims := &models.UserClaims{
						UserID:           "STU1001",
						Role:             "student",
						ClassID:          "10",
						SectionID:        "A",
						LinkedStudentIDs: []string{"STU1001"},
					}
					ctx := context.WithValue(r.Context(), userClaimsKey{}, defaultClaims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"success":false,"error":"unauthorized_missing_token"}`))
				return
			}

			claims := &models.UserClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				// Fall back to student claims compatibility
				stuClaims := &models.StudentClaims{}
				stuToken, sErr := jwt.ParseWithClaims(tokenStr, stuClaims, func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, jwt.ErrSignatureInvalid
					}
					return []byte(jwtSecret), nil
				})
				if sErr == nil && stuToken.Valid {
					claims = &models.UserClaims{
						UserID:           stuClaims.StudentID,
						Role:             "student",
						ClassID:          stuClaims.Grade,
						SectionID:        stuClaims.Section,
						LinkedStudentIDs: []string{stuClaims.StudentID},
					}
				} else {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"success":false,"error":"unauthorized_invalid_token"}`))
					return
				}
			}

			ctx := context.WithValue(r.Context(), userClaimsKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserClaims retrieves the UserClaims from the context. Returns nil if not authenticated.
func GetUserClaims(ctx context.Context) *models.UserClaims {
	if val := ctx.Value(userClaimsKey{}); val != nil {
		if c, ok := val.(*models.UserClaims); ok {
			return c
		}
	}
	return nil
}
