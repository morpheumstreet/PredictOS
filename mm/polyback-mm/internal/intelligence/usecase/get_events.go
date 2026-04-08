package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/dome"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/kalshidflow"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/gamma"
)

// GetEvents implements the former get-events edge-function contract.
type GetEvents struct {
	root *config.Root
	hc   *http.Client
}

func NewGetEvents(root *config.Root, hc *http.Client) *GetEvents {
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	return &GetEvents{root: root, hc: hc}
}

type getEventsReq struct {
	URL string `json:"url"`
}

// Run executes get-events logic; ctx is respected for outbound HTTP.
func (g *GetEvents) Run(ctx context.Context, body []byte) (status int, out map[string]any) {
	start := time.Now()
	meta := func() map[string]any {
		return map[string]any{
			"requestId":        fmt.Sprintf("%d", time.Now().UnixNano()),
			"timestamp":        time.Now().UTC().Format(time.RFC3339Nano),
			"processingTimeMs": time.Since(start).Milliseconds(),
		}
	}
	var req getEventsReq
	if err := json.Unmarshal(body, &req); err != nil {
		return 400, map[string]any{"success": false, "error": "Invalid JSON in request body", "metadata": meta()}
	}
	url := strings.TrimSpace(req.URL)
	if url == "" {
		return 400, map[string]any{"success": false, "error": "Missing required parameter: 'url'", "metadata": meta()}
	}

	pmType, urlSource, ok := detectPmTypeAndSource(url)
	if !ok {
		return 400, map[string]any{"success": false, "error": "Could not detect prediction market type from URL. Use Kalshi, Polymarket, or Jupiter prediction market URLs.", "metadata": meta()}
	}
	dataProvider := "dflow"
	if pmType == "Polymarket" {
		dataProvider = "dome" // logical label in TS for routing; Polymarket uses gamma
	}

	var eventIdentifier string
	var eventID string
	var markets []any

	switch pmType {
	case "Kalshi":
		ticker, err := extractKalshiTicker(url, urlSource)
		if err != nil {
			return 400, map[string]any{"success": false, "error": err.Error(), "metadata": meta()}
		}
		eventIdentifier = ticker
		base := strings.TrimSpace(g.root.Hft.KalshiDFlow.BaseURL)
		key := strings.TrimSpace(g.root.Hft.KalshiDFlow.APIKey)
		if key == "" {
			key = strings.TrimSpace(os.Getenv("DFLOW_API_KEY"))
		}
		if base == "" {
			base = strings.TrimSpace(os.Getenv("DFLOW_BASE_URL"))
		}
		var marketsErr error
		if key != "" {
			raw, ferr := kalshidflow.FetchMarketsByEventTicker(ctx, base, key, ticker)
			if ferr == nil {
				arr, perr := kalshidflow.ParseMarketsArray(raw)
				if perr != nil {
					marketsErr = perr
				} else {
					for _, m := range arr {
						var v any
						_ = json.Unmarshal(m, &v)
						markets = append(markets, v)
					}
				}
			} else {
				marketsErr = ferr
			}
		}
		if len(markets) == 0 && marketsErr == nil {
			marketsErr = fmt.Errorf("DFlow API key not configured")
		}
		if len(markets) == 0 {
			if alt, derr := g.fetchKalshiViaDome(ctx, ticker); derr == nil && len(alt) > 0 {
				markets = alt
				marketsErr = nil
			}
		}
		if len(markets) == 0 && marketsErr != nil {
			return dflowOrKalshiError(marketsErr, ticker, meta)
		}
	case "Polymarket":
		slug := extractPolymarketSlug(url)
		if slug == "" {
			return 400, map[string]any{"success": false, "error": "Could not extract event slug from URL", "metadata": meta()}
		}
		eventIdentifier = slug
		gc := gamma.New(strings.TrimSpace(g.root.Hft.Polymarket.GammaURL))
		if strings.TrimSpace(g.root.Hft.Polymarket.GammaURL) == "" {
			gc = gamma.New("https://gamma-api.polymarket.com")
		}
		raw, err := gc.EventsBySlug(slug)
		if err != nil {
			raw, err = gc.EventBySlugPath(slug)
		}
		if err != nil {
			return polyNotFoundOr502(err, slug, meta)
		}
		eventID, markets = parseGammaEventMarkets(raw)
	}

	if len(markets) == 0 {
		return 404, map[string]any{"success": false, "error": fmt.Sprintf("No markets found for '%s' on %s.", eventIdentifier, pmType), "metadata": meta()}
	}

	effectiveProvider := dataProvider
	if pmType == "Polymarket" {
		effectiveProvider = "gamma"
	}

	return 200, map[string]any{
		"success":          true,
		"eventIdentifier":  eventIdentifier,
		"eventId":          nullIfEmpty(eventID),
		"pmType":           pmType,
		"urlSource":        urlSource,
		"markets":          markets,
		"marketsCount":     len(markets),
		"dataProvider":     effectiveProvider,
		"metadata":         meta(),
	}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func detectPmTypeAndSource(rawURL string) (pmType string, urlSource string, ok bool) {
	u := strings.ToLower(rawURL)
	switch {
	case strings.Contains(u, "jup.ag/prediction"):
		return "Kalshi", "jupiter", true
	case strings.Contains(u, "kalshi"):
		return "Kalshi", "kalshi", true
	case strings.Contains(u, "polymarket"):
		return "Polymarket", "polymarket", true
	default:
		return "", "", false
	}
}

func extractJupiterTicker(url string) string {
	parts := strings.Split(strings.Split(url, "?")[0], "/")
	if len(parts) == 0 {
		return ""
	}
	t := strings.TrimSpace(parts[len(parts)-1])
	if t == "" {
		return ""
	}
	return strings.ToUpper(t)
}

func extractKalshiTicker(url, urlSource string) (string, error) {
	if urlSource == "jupiter" {
		t := extractJupiterTicker(url)
		if t == "" {
			return "", fmt.Errorf("could not extract event ticker from URL")
		}
		return t, nil
	}
	parts := strings.Split(strings.Split(url, "?")[0], "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("could not extract event ticker from URL")
	}
	return strings.ToUpper(parts[len(parts)-1]), nil
}

func extractPolymarketSlug(url string) string {
	parts := strings.Split(strings.Split(url, "?")[0], "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func parseGammaEventMarkets(raw json.RawMessage) (eventID string, markets []any) {
	var asArray []map[string]any
	if json.Unmarshal(raw, &asArray) == nil && len(asArray) > 0 {
		return stringifyID(asArray[0]["id"]), marketsFromEvent(asArray[0])
	}
	var one map[string]any
	if json.Unmarshal(raw, &one) == nil {
		return stringifyID(one["id"]), marketsFromEvent(one)
	}
	return "", nil
}

func stringifyID(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(t)
	}
}

func marketsFromEvent(ev map[string]any) []any {
	m, _ := ev["markets"].([]any)
	return m
}

func dflowOrKalshiError(err error, ticker string, meta func() map[string]any) (int, map[string]any) {
	msg := err.Error()
	notFound := strings.Contains(msg, "404") || strings.Contains(strings.ToLower(msg), "not found")
	code := 502
	if notFound {
		code = 404
		msg = fmt.Sprintf("Event '%s' not found on Kalshi (via DFlow).", ticker)
	}
	return code, map[string]any{"success": false, "error": msg, "metadata": meta()}
}

func polyNotFoundOr502(err error, slug string, meta func() map[string]any) (int, map[string]any) {
	msg := err.Error()
	notFound := strings.Contains(msg, "404") || strings.Contains(strings.ToLower(msg), "not found")
	if notFound {
		return 404, map[string]any{"success": false, "error": fmt.Sprintf("Event '%s' not found on Polymarket.", slug), "metadata": meta()}
	}
	return 502, map[string]any{"success": false, "error": fmt.Sprintf("Failed to fetch markets from Polymarket: %s", msg), "metadata": meta()}
}

// fetchKalshiViaDome is optional when DFlow fails — skip unless DOME_API_KEY set
func (g *GetEvents) fetchKalshiViaDome(ctx context.Context, ticker string) ([]any, error) {
	dc, err := dome.NewFromEnv()
	if err != nil {
		return nil, err
	}
	raw, err := dc.KalshiMarketsByEvent(ctx, ticker, "open", 100)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(raw))
	for _, m := range raw {
		var v any
		_ = json.Unmarshal(m, &v)
		out = append(out, v)
	}
	return out, nil
}
