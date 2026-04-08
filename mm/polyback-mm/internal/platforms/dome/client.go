// Package dome is a minimal HTTP client for Dome API (Kalshi market lists).
// See https://docs.domeapi.io/
package dome

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const defaultBase = "https://api.domeapi.io/v1"

// Client calls Dome REST with bearer auth.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewClient(apiKey, baseURL string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBase
	}
	return &Client{
		baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// NewFromEnv uses DOME_API_KEY and optional DOME_BASE_URL.
func NewFromEnv() (*Client, error) {
	k := strings.TrimSpace(os.Getenv("DOME_API_KEY"))
	if k == "" {
		return nil, fmt.Errorf("DOME_API_KEY is not set")
	}
	return NewClient(k, os.Getenv("DOME_BASE_URL")), nil
}

// KalshiMarketsByEvent fetches GET /kalshi/markets with event_ticker and status.
func (c *Client) KalshiMarketsByEvent(ctx context.Context, eventTicker, status string, limit int) ([]json.RawMessage, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("dome: missing api key")
	}
	if limit <= 0 {
		limit = 100
	}
	if status == "" {
		status = "open"
	}
	u, err := url.Parse(c.baseURL + "/kalshi/markets")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("event_ticker", eventTicker)
	q.Set("status", status)
	q.Set("limit", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dome kalshi/markets: status %d body=%s", resp.StatusCode, string(b))
	}
	var out struct {
		Markets []json.RawMessage `json:"markets"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out.Markets, nil
}
