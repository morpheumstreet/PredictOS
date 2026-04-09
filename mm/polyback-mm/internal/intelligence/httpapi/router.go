package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/adapters/polyfactual"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/app"
)

// Mount registers /api/intelligence routes on r.
func Mount(r chi.Router, d *app.Deps) {
	r.Post("/ping", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, 200, map[string]any{"ok": true})
	})

	r.Post("/polyfactual-research", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		var in map[string]any
		if err := json.Unmarshal(body, &in); err != nil {
			WriteJSON(w, 400, map[string]any{"success": false, "error": "Invalid JSON"})
			return
		}
		q, _ := in["query"].(string)
		text := true
		if v, ok := in["text"].(bool); ok {
			text = v
		}
		start := time.Now()
		out, err := d.Polyfactual.GenerateAnswer(polyfactual.AnswerRequest{Query: q, Text: &text})
		if err != nil {
			WriteJSON(w, 500, map[string]any{
				"success": false,
				"error":   err.Error(),
				"metadata": map[string]any{
					"requestId":          time.Now().Format(time.RFC3339Nano),
					"timestamp":          time.Now().UTC().Format(time.RFC3339Nano),
					"processingTimeMs":   time.Since(start).Milliseconds(),
					"query":              q,
				},
			})
			return
		}
		out["metadata"] = map[string]any{
			"requestId":        time.Now().Format(time.RFC3339Nano),
			"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
			"processingTimeMs": time.Since(start).Milliseconds(),
			"query":            q,
		}
		WriteJSON(w, 200, out)
	})

	r.Post("/x402-seller", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		var in map[string]any
		if err := json.Unmarshal(body, &in); err != nil {
			WriteJSON(w, 400, map[string]any{"success": false, "error": "Invalid JSON"})
			return
		}
		st, out := d.X402.HandlePOST(in)
		WriteJSON(w, st, out)
	})

	r.Post("/get-events", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		st, out := d.GetEvents.Run(req.Context(), body)
		WriteJSON(w, st, out)
	})

	r.Post("/event-analysis-agent", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		ctx, cancel := context.WithTimeout(req.Context(), 180*time.Second)
		defer cancel()
		st, out := d.EventAnalysis.Run(ctx, body)
		WriteJSON(w, st, out)
	})

	r.Post("/analyze-event-markets", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		ctx, cancel := context.WithTimeout(req.Context(), 180*time.Second)
		defer cancel()
		st, out := d.AnalyzeMarkets.Run(ctx, body)
		WriteJSON(w, st, out)
	})

	r.Post("/bookmaker-agent", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		ctx, cancel := context.WithTimeout(req.Context(), 180*time.Second)
		defer cancel()
		st, out := d.Bookmaker.Run(ctx, body)
		WriteJSON(w, st, out)
	})

	r.Post("/arbitrage-finder", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		ctx, cancel := context.WithTimeout(req.Context(), 180*time.Second)
		defer cancel()
		st, out := d.ArbitrageFinder.Run(ctx, body)
		WriteJSON(w, st, out)
	})

	r.Post("/mapper-agent", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		st, out := d.Mapper.Run(body)
		WriteJSON(w, st, out)
	})

	r.Post("/polymarket-put-order", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		st, out := d.Trading.PolymarketPutOrder(req.Context(), body)
		WriteJSON(w, st, out)
	})

	r.Post("/polymarket-up-down-15-markets-limit-order-bot", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		st, out := d.Trading.LimitOrderBot(req.Context(), body)
		WriteJSON(w, st, out)
	})

	r.Post("/polymarket-position-tracker", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		st, out := d.Trading.PositionTracker(req.Context(), body)
		WriteJSON(w, st, out)
	})

	r.Post("/alpha-rules/collect", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		ctx, cancel := context.WithTimeout(req.Context(), 600*time.Second)
		defer cancel()
		st, out := d.AlphaRules.RunCollect(ctx, body)
		WriteJSON(w, st, out)
	})

	r.Post("/alpha-rules/description-agent", func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		ctx, cancel := context.WithTimeout(req.Context(), 600*time.Second)
		defer cancel()
		st, out := d.AlphaRules.RunDescriptionAgent(ctx, body)
		WriteJSON(w, st, out)
	})
}
