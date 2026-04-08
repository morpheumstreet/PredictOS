package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/llm"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/prompts"
)

type EventAnalysis struct {
	llm *llm.Facade
}

func NewEventAnalysis(f *llm.Facade) *EventAnalysis {
	return &EventAnalysis{llm: f}
}

func (e *EventAnalysis) Run(ctx context.Context, body []byte) (int, map[string]any) {
	start := time.Now()
	meta := func(extra map[string]any) map[string]any {
		m := map[string]any{
			"requestId":        fmt.Sprintf("%d", time.Now().UnixNano()),
			"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
			"processingTimeMs": time.Since(start).Milliseconds(),
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}

	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return 400, map[string]any{"success": false, "error": "Invalid JSON in request body"}
	}
	markets, _ := req["markets"].([]any)
	eventID, _ := req["eventIdentifier"].(string)
	pmType, _ := req["pmType"].(string)
	model, _ := req["model"].(string)
	question, _ := req["question"].(string)
	var tools []string
	if ta, ok := req["tools"].([]any); ok {
		for _, t := range ta {
			if s, ok := t.(string); ok {
				tools = append(tools, s)
			}
		}
	}
	userCommand, _ := req["userCommand"].(string)

	if len(markets) == 0 {
		return 400, map[string]any{"success": false, "error": "Missing or invalid 'markets' parameter"}
	}
	if eventID == "" {
		return 400, map[string]any{"success": false, "error": "Missing required parameter: 'eventIdentifier'"}
	}
	if pmType != "Kalshi" && pmType != "Polymarket" {
		return 400, map[string]any{"success": false, "error": "Invalid 'pmType'. Must be 'Kalshi' or 'Polymarket'"}
	}
	if model == "" {
		return 400, map[string]any{"success": false, "error": "Missing required parameter: 'model'"}
	}
	if question == "" {
		question = "What is the best trading opportunity in this market? Analyze the probability and provide a recommendation."
	}

	sys, usr := prompts.AnalyzeEventMarkets(markets, eventID, question, pmType, tools, userCommand)
	text, respModel, tokens, payCost, err := e.llm.CompleteJSON(ctx, model, sys, usr, tools)
	if err != nil {
		return 500, map[string]any{"success": false, "error": err.Error(), "metadata": meta(map[string]any{"model": model})}
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return 500, map[string]any{"success": false, "error": "Failed to parse AI response as JSON", "metadata": meta(map[string]any{"model": respModel, "tokensUsed": tokens})}
	}
	md := meta(map[string]any{"model": respModel, "tokensUsed": tokens})
	if payCost != nil {
		md["paymentCost"] = *payCost
	}
	return 200, map[string]any{"success": true, "data": data, "metadata": md}
}
