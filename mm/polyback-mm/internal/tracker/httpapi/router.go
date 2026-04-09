package httpapi

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker"
)

// Deps holds tracker HTTP dependencies.
type Deps struct {
	Svc *tracker.Service
}

// Mount registers /api/tracker routes on r.
func Mount(r chi.Router, d *Deps) {
	if d == nil || d.Svc == nil {
		return
	}
	r.Post("/polymarket-position-tracker", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		st, out := d.Svc.PolymarketPositionTracker(req.Context(), body)
		writeJSON(w, st, out)
	})
}

// WriteJSON writes v as JSON with status (mirrors intelligence httpapi).
func writeJSON(w http.ResponseWriter, status int, v any) {
	if status == 0 {
		status = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
