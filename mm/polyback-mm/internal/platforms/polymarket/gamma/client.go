package gamma

import (
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
	return &Client{
		baseURL: strings.TrimSuffix(strings.TrimSpace(base), "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
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
