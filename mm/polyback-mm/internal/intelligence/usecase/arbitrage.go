package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/arbitragefee"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/llm"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/prompts"
)

// ArbitrageFinder is a streamlined port: fetch source event, search other venue via DFlow text search, LLM analysis.
type ArbitrageFinder struct {
	root *config.Root
	hc   *http.Client
	llm  *llm.Facade
	ge   *GetEvents
}

func NewArbitrageFinder(root *config.Root, hc *http.Client, f *llm.Facade) *ArbitrageFinder {
	if hc == nil {
		hc = &http.Client{Timeout: 120 * time.Second}
	}
	return &ArbitrageFinder{root: root, hc: hc, llm: f, ge: NewGetEvents(root, hc)}
}

func (a *ArbitrageFinder) Run(ctx context.Context, body []byte) (int, map[string]any) {
	start := time.Now()
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return 400, map[string]any{"success": false, "error": "Invalid JSON"}
	}
	rawURL, _ := req["url"].(string)
	model, _ := req["model"].(string)
	if rawURL == "" || model == "" {
		return 400, map[string]any{"success": false, "error": "Missing url or model"}
	}

	st, geOut := a.ge.Run(ctx, mustJSON(map[string]any{"url": rawURL}))
	if st != 200 {
		return st, geOut
	}
	pmType, _ := geOut["pmType"].(string)
	markets, _ := geOut["markets"].([]any)
	source := "polymarket"
	other := "kalshi"
	if pmType == "Kalshi" {
		source = "kalshi"
		other = "polymarket"
	}

	// Short search query from first market title
	title := eventTitleHint(markets)
	otherMarkets := a.searchOther(ctx, other, title)

	sys, usr := prompts.ArbitrageAnalysis(source, markets, other, otherMarkets, title)
	text, respModel, tokens, _, err := a.llm.CompleteJSON(ctx, model, sys, usr, nil)
	if err != nil {
		return 500, map[string]any{"success": false, "error": err.Error()}
	}
	var analysis map[string]any
	if err := json.Unmarshal([]byte(text), &analysis); err != nil {
		return 500, map[string]any{"success": false, "error": "parse analysis json"}
	}
	if arb, ok := analysis["arbitrage"].(map[string]any); ok {
		arbitragefee.Enrich(arb, arbitragefee.LoadConfig())
		analysis["arbitrage"] = arb
	}

	meta := map[string]any{
		"requestId":          fmt.Sprintf("%d", time.Now().UnixNano()),
		"timestamp":          time.Now().UTC().Format(time.RFC3339Nano),
		"processingTimeMs":   time.Since(start).Milliseconds(),
		"model":              respModel,
		"tokensUsed":         tokens,
		"sourceMarket":       source,
		"searchedMarket":     other,
	}
	return 200, map[string]any{"success": true, "data": analysis, "metadata": meta}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func eventTitleHint(markets []any) string {
	if len(markets) == 0 {
		return ""
	}
	m, ok := markets[0].(map[string]any)
	if !ok {
		return ""
	}
	if t, ok := m["title"].(string); ok {
		return t
	}
	if t, ok := m["question"].(string); ok {
		return t
	}
	return ""
}

// searchOther runs a minimal Gamma tag search for polymarket or DFlow search for kalshi.
func (a *ArbitrageFinder) searchOther(ctx context.Context, platform, q string) []any {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil
	}
	if platform == "polymarket" {
		qlen := len(q)
		if qlen > 80 {
			qlen = 80
		}
		u := "https://gamma-api.polymarket.com/public-search?q=" + url.QueryEscape(q[:qlen])
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		resp, err := a.hc.Do(req)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		var wrap map[string]any
		if json.Unmarshal(b, &wrap) != nil {
			return nil
		}
		if ev, ok := wrap["events"].([]any); ok {
			return ev
		}
		if mk, ok := wrap["markets"].([]any); ok {
			return mk
		}
		return nil
	}
	// Kalshi-shaped search via DFlow (best-effort)
	base := strings.TrimSpace(a.root.Hft.KalshiDFlow.BaseURL)
	key := strings.TrimSpace(a.root.Hft.KalshiDFlow.APIKey)
	if key == "" {
		return nil
	}
	searchURL := base + "/search?q=" + url.QueryEscape(q)
	if !strings.Contains(base, "://") {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("x-api-key", key)
	resp, err := a.hc.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	var out map[string]any
	if json.Unmarshal(b, &out) != nil {
		return nil
	}
	if r, ok := out["results"].([]any); ok {
		return r
	}
	return nil
}
