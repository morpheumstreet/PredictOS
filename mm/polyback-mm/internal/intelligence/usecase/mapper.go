package usecase

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/mapping"
)

type Mapper struct{}

func NewMapper() *Mapper { return &Mapper{} }

func (m *Mapper) Run(body []byte) (int, map[string]any) {
	start := time.Now()
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return 400, map[string]any{"success": false, "error": "Invalid JSON"}
	}
	platform, _ := req["platform"].(string)
	if platform != "Polymarket" && platform != "Kalshi" {
		return 400, map[string]any{"success": false, "error": "Invalid platform. Must be 'Polymarket' or 'Kalshi'"}
	}
	if platform == "Kalshi" {
		return 501, map[string]any{
			"success": false,
			"error":   "Kalshi autonomous mode coming soon! Currently only Polymarket is supported.",
			"metadata": map[string]any{
				"requestId":          fmt.Sprintf("%d", time.Now().UnixNano()),
				"timestamp":          time.Now().UTC().Format(time.RFC3339Nano),
				"processingTimeMs":   time.Since(start).Milliseconds(),
			},
		}
	}
	araw, ok := req["analysisResult"].(map[string]any)
	if !ok {
		return 400, map[string]any{"success": false, "error": "Missing analysisResult"}
	}
	rec, _ := araw["recommendedAction"].(string)
	if rec == "NO TRADE" {
		return 200, map[string]any{
			"success": false,
			"error":   "Agents recommend NO TRADE - no order to place",
			"metadata": map[string]any{
				"requestId":        fmt.Sprintf("%d", time.Now().UnixNano()),
				"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
				"processingTimeMs": time.Since(start).Milliseconds(),
			},
		}
	}
	md, ok := req["marketData"].(map[string]any)
	if !ok {
		return 400, map[string]any{"success": false, "error": "Missing marketData"}
	}
	budget, ok := req["budgetUsd"].(float64)
	if !ok {
		return 400, map[string]any{"success": false, "error": "Invalid budgetUsd"}
	}
	if err := mapping.ValidateBudget(budget); err != nil {
		return 400, map[string]any{"success": false, "error": err.Error()}
	}

	mkt := mapping.MarketData{
		Title:           str(md["title"]),
		Question:        str(md["question"]),
		Slug:            str(md["slug"]),
		ConditionID:     str(md["conditionId"]),
		ClobTokenIds:    str(md["clobTokenIds"]),
		Outcomes:        str(md["outcomes"]),
		OutcomePrices:   str(md["outcomePrices"]),
		MinimumTickSize: str(md["minimumTickSize"]),
		Closed:          boolField(md["closed"]),
	}
	if v, ok := md["acceptingOrders"].(bool); ok {
		mkt.AcceptingOrders = &v
	}
	if v, ok := md["negRisk"].(bool); ok {
		mkt.NegRisk = &v
	}
	analysis := mapping.AnalysisResult{RecommendedAction: rec}
	params, err := mapping.MapPolymarketOrder(analysis, mkt, budget)
	if err != nil {
		return 400, map[string]any{"success": false, "error": err.Error()}
	}
	return 200, map[string]any{
		"success": true,
		"data": map[string]any{
			"platform":     "Polymarket",
			"orderParams":  params,
			"humanSummary": params.OrderDescription,
		},
		"metadata": map[string]any{
			"requestId":        fmt.Sprintf("%d", time.Now().UnixNano()),
			"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
			"processingTimeMs": time.Since(start).Milliseconds(),
		},
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func boolField(v any) bool {
	b, _ := v.(bool)
	return b
}
