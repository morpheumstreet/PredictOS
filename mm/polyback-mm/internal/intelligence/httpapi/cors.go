package httpapi

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// UseIntelligenceCORS allows browser POST to /api/intelligence when global server CORS is GET-only.
func UseIntelligenceCORS(r chi.Router, allowedOrigins []string) {
	if len(allowedOrigins) == 0 {
		return
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "HEAD", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "apikey", "x-client-info"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
}
