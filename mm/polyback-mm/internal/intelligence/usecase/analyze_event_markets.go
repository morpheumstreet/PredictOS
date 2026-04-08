package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/llm"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/prompts"
)

// AnalyzeEventMarkets composes get-events + event-analysis style LLM call.
type AnalyzeEventMarkets struct {
	events *GetEvents
	llm    *llm.Facade
}

func NewAnalyzeEventMarkets(ge *GetEvents, f *llm.Facade) *AnalyzeEventMarkets {
	return &AnalyzeEventMarkets{events: ge, llm: f}
}

func (a *AnalyzeEventMarkets) Run(ctx context.Context, body []byte) (int, map[string]any) {
	start := time.Now()
	meta := func(m map[string]any) map[string]any {
		if m == nil {
			m = map[string]any{}
		}
		m["requestId"] = fmt.Sprintf("%d", time.Now().UnixNano())
		m["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
		m["processingTimeMs"] = time.Since(start).Milliseconds()
		return m
	}

	var wrap map[string]any
	if err := json.Unmarshal(body, &wrap); err != nil {
		return 400, map[string]any{"success": false, "error": "Invalid JSON", "metadata": meta(nil)}
	}
	url, _ := wrap["url"].(string)
	model, _ := wrap["model"].(string)
	question, _ := wrap["question"].(string)
	if url == "" {
		return 400, map[string]any{"success": false, "error": "Missing url", "metadata": meta(nil)}
	}
	if model == "" {
		return 400, map[string]any{"success": false, "error": "Missing model", "metadata": meta(nil)}
	}

	geBody, _ := json.Marshal(map[string]any{"url": url})
	st, geResp := a.events.Run(ctx, geBody)
	if st != 200 {
		return st, geResp
	}
	markets, _ := geResp["markets"].([]any)
	eventID, _ := geResp["eventIdentifier"].(string)
	pmType, _ := geResp["pmType"].(string)
	if question == "" {
		question = "What is the best trading opportunity across these markets?"
	}
	var tools []string
	if ta, ok := wrap["tools"].([]any); ok {
		for _, t := range ta {
			if s, ok := t.(string); ok {
				tools = append(tools, s)
			}
		}
	}
	userCmd, _ := wrap["userCommand"].(string)

	sys, usr := prompts.AnalyzeEventMarkets(markets, eventID, question, pmType, tools, userCmd)
	text, respModel, tokens, payCost, err := a.llm.CompleteJSON(ctx, model, sys, usr, tools)
	if err != nil {
		return 500, map[string]any{"success": false, "error": err.Error(), "metadata": meta(map[string]any{"model": model})}
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return 500, map[string]any{"success": false, "error": "Failed to parse AI response as JSON", "metadata": meta(map[string]any{"model": respModel})}
	}
	md := meta(map[string]any{"model": respModel, "tokensUsed": tokens})
	if payCost != nil {
		md["paymentCost"] = *payCost
	}
	return 200, map[string]any{"success": true, "data": data, "metadata": md, "eventContext": map[string]any{"eventIdentifier": eventID, "pmType": pmType, "marketsCount": len(markets)}}
}
