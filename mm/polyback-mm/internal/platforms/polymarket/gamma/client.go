package gamma

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(base string) *Client {
	return NewWithHTTP(base, nil)
}

// NewWithHTTP builds a Gamma client. When hc is nil, a 15s timeout client is used.
func NewWithHTTP(base string, hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimSuffix(strings.TrimSpace(base), "/"),
		httpClient: hc,
	}
}

// Markets fetches GET /markets with query params (e.g. clob_token_ids).
func (c *Client) Markets(query map[string]string) (json.RawMessage, error) {
	u, err := url.Parse(c.baseURL + "/markets")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gamma /markets: status %d body=%s", resp.StatusCode, string(b))
	}
	return json.RawMessage(b), nil
}

// EventsBySlug fetches GET /events?slug=...
func (c *Client) EventsBySlug(slug string) (json.RawMessage, error) {
	u := c.baseURL + "/events?slug=" + url.QueryEscape(slug)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gamma /events: status %d body=%s", resp.StatusCode, string(b))
	}
	return json.RawMessage(b), nil
}

// FetchEventsPage calls GET /events with list-style query params (pagination).
// Matches Polymarket Gamma usage in strat/alpha-rules/collect.py.
func (c *Client) FetchEventsPage(ctx context.Context, limit, offset int, active, closed, archived bool) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("active", fmt.Sprintf("%t", active))
	q.Set("closed", fmt.Sprintf("%t", closed))
	q.Set("archived", fmt.Sprintf("%t", archived))
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))
	u := c.baseURL + "/events?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "PredictOS-polyback-intelligence/1.0")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gamma /events list: status %d body=%s", resp.StatusCode, string(b))
	}
	return json.RawMessage(b), nil
}
