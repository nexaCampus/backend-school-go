package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

// NewCORS creates a CORS middleware configured with allowed origins and headers.
func NewCORS(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := allowedOrigins
	if len(origins) == 0 || (len(origins) == 1 && origins[0] == "*") {
		origins = []string{"*"}
	}

	return cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
		ExposedHeaders:   []string{"Link", "Content-Length"},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	})
}
