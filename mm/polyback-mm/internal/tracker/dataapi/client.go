package dataapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/port"
)

// Client calls Polymarket data-api (public HTTPS).
type Client struct {
	base string
	hc   *http.Client
}

var _ port.PolymarketData = (*Client)(nil)

// New returns a client for baseURL (e.g. https://data-api.polymarket.com). hc may be nil.
func New(baseURL string, hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		base: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		hc:   hc,
	}
}

// Positions implements port.PolymarketData.
func (c *Client) Positions(ctx context.Context, user string) ([]map[string]any, error) {
	u := fmt.Sprintf("%s/positions?user=%s&sizeThreshold=0", c.base, url.QueryEscape(user))
	return c.getJSONArray(ctx, u, "positions")
}

// Activity implements port.PolymarketData.
func (c *Client) Activity(ctx context.Context, user string) ([]map[string]any, error) {
	u := fmt.Sprintf("%s/activity?user=%s&limit=500", c.base, url.QueryEscape(user))
	return c.getJSONArray(ctx, u, "activity")
}

func (c *Client) getJSONArray(ctx context.Context, fullURL, endpoint string) ([]map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("data-api %s: %s", endpoint, resp.Status)
	}
	var out []map[string]any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("data-api %s json: %w", endpoint, err)
	}
	return out, nil
}
