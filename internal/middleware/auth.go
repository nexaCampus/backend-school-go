package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nexaCampus/backend-school-go/internal/models"
)

type userClaimsKey struct{}

// JWTAuth verifies Bearer tokens and attaches UserClaims to request context.
func JWTAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenStr string
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			} else if qToken := r.URL.Query().Get("token"); qToken != "" {
				tokenStr = qToken
			}

			if tokenStr == "" {
				// Provide default student context for easy developer testing and portal browsing
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

			claims := &models.UserClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				// Fall back to student claims compatibility
				stuClaims := &models.StudentClaims{}
				stuToken, sErr := jwt.ParseWithClaims(tokenStr, stuClaims, func(t *jwt.Token) (interface{}, error) {
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
					http.Error(w, `{"success":false,"error":"unauthorized_invalid_token"}`, http.StatusUnauthorized)
					return
				}
			}

			ctx := context.WithValue(r.Context(), userClaimsKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserClaims retrieves the UserClaims from the context.
func GetUserClaims(ctx context.Context) *models.UserClaims {
	if val := ctx.Value(userClaimsKey{}); val != nil {
		if c, ok := val.(*models.UserClaims); ok {
			return c
		}
	}
	return &models.UserClaims{
		UserID:           "STU1001",
		Role:             "student",
		ClassID:          "10",
		SectionID:        "A",
		LinkedStudentIDs: []string{"STU1001"},
	}
}
