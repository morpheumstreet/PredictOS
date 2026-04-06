package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/gabagool"
)

func Mount(r chi.Router, eng *gabagool.Engine, enabled bool) {
	r.Get("/api/strategy/status", func(w http.ResponseWriter, _ *http.Request) {
		s := struct {
			ActiveMarkets      int  `json:"activeMarkets"`
			Running            bool `json:"running"`
			MarketMakerEnabled bool `json:"marketMakerEnabled"`
		}{
			ActiveMarkets:      eng.ActiveMarketCount(),
			Running:            enabled,
			MarketMakerEnabled: eng.MarketMakerEnabled(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s)
	})
}
