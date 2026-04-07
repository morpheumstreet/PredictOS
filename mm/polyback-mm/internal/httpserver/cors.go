package httpserver

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// UseCORSIfConfigured registers chi CORS middleware when cors_allowed_origins is non-empty
// in YAML (server.*). Required for browser calls to polyback-mm without the Bun relay.
func UseCORSIfConfigured(r chi.Router, allowedOrigins []string) {
	if len(allowedOrigins) == 0 {
		return
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
}
