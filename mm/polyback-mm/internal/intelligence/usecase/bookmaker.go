package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/llm"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/prompts"
)

type Bookmaker struct {
	llm *llm.Facade
}

func NewBookmaker(f *llm.Facade) *Bookmaker {
	return &Bookmaker{llm: f}
}

func (b *Bookmaker) Run(ctx context.Context, body []byte) (int, map[string]any) {
	start := time.Now()
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return 400, map[string]any{"success": false, "error": "Invalid JSON"}
	}
	model, _ := req["model"].(string)
	eventID, _ := req["eventIdentifier"].(string)
	pmType, _ := req["pmType"].(string)
	if model == "" || eventID == "" || pmType == "" {
		return 400, map[string]any{"success": false, "error": "Missing model, eventIdentifier, or pmType"}
	}
	var analyses []map[string]any
	if a, ok := req["analyses"].([]any); ok {
		for _, x := range a {
			if m, ok := x.(map[string]any); ok {
				analyses = append(analyses, m)
			}
		}
	}
	var x402 []map[string]any
	if a, ok := req["x402Results"].([]any); ok {
		for _, x := range a {
			if m, ok := x.(map[string]any); ok {
				x402 = append(x402, m)
			}
		}
	}
	if len(analyses)+len(x402) < 2 {
		return 400, map[string]any{"success": false, "error": "Need at least two analyses or x402Results"}
	}
	sys, usr := prompts.Bookmaker(analyses, x402, eventID, pmType)
	text, respModel, tokens, pay, err := b.llm.CompleteJSON(ctx, model, sys, usr, nil)
	if err != nil {
		return 500, map[string]any{"success": false, "error": err.Error()}
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return 500, map[string]any{"success": false, "error": "Failed to parse bookmaker JSON"}
	}
	md := map[string]any{
		"requestId":        fmt.Sprintf("%d", time.Now().UnixNano()),
		"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
		"processingTimeMs": time.Since(start).Milliseconds(),
		"model":            respModel,
		"tokensUsed":       tokens,
	}
	if pay != nil {
		md["paymentCost"] = *pay
	}
	return 200, map[string]any{"success": true, "data": data, "metadata": md}
}
