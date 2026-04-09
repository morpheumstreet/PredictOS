package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/profitlock/PredictOS/mm/irys-upload/internal/config"
	"github.com/profitlock/PredictOS/mm/irys-upload/internal/irysupload"
)

type Server struct {
	cfg    *config.Config
	client *irysupload.Client
}

func New(cfg *config.Config) *Server {
	return &Server{
		cfg:    cfg,
		client: irysupload.NewClient(cfg),
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(180 * time.Second))

	r.Get("/status", s.handleStatus)
	r.Post("/upload", s.handleUpload)
	return r
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.client.Status())
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 52<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "read body: " + err.Error()})
		return
	}
	ctx := r.Context()
	res, err := s.client.Upload(ctx, body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	if !res.Success {
		writeJSON(w, http.StatusBadRequest, res)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"transactionId": res.TransactionID,
		"gatewayUrl":    res.GatewayURL,
		"environment":   res.Environment,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Health is a minimal readiness handler without loading full config (for tests).
func Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
