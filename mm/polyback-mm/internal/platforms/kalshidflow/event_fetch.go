package kalshidflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms"
)

// FetchMarketsByEventTicker returns the raw JSON body from DFlow GET /event/{ticker}?withNestedMarkets=true.
func FetchMarketsByEventTicker(ctx context.Context, baseURL, apiKey, eventTicker string) ([]byte, error) {
	eventTicker = strings.TrimSpace(eventTicker)
	apiKey = strings.TrimSpace(apiKey)
	if eventTicker == "" {
		return nil, fmt.Errorf("kalshidflow: empty event ticker")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("kalshidflow: empty api key")
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	u := fmt.Sprintf("%s/event/%s?withNestedMarkets=true", baseURL, url.PathEscape(eventTicker))
	return platforms.DoJSONExpect2xx(ctx, platforms.DefaultHTTPClient(), http.MethodGet, u, map[string]string{"x-api-key": apiKey}, nil)
}

// ParseMarketsArray extracts the "markets" array from a DFlow event JSON body (may be empty on error shape).
func ParseMarketsArray(body []byte) ([]json.RawMessage, error) {
	var wrap struct {
		Markets []json.RawMessage `json:"markets"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, err
	}
	return wrap.Markets, nil
}
