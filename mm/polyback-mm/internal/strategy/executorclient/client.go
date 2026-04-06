package executorclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/api"
	"github.com/shopspring/decimal"
)

type Client struct {
	base    string
	client  *http.Client
	liveAck bool
}

func New(root *config.Root) *Client {
	return &Client{
		base:    strings.TrimSuffix(root.Hft.Executor.BaseURL, "/"),
		client:  &http.Client{Timeout: 8 * time.Second},
		liveAck: root.Hft.Executor.SendLiveAck,
	}
}

func (c *Client) GetTickSize(tokenID string) (decimal.Decimal, error) {
	var d decimal.Decimal
	err := c.getJSON("/api/polymarket/tick-size/"+url.PathEscape(tokenID), &d)
	return d, err
}

func (c *Client) PlaceLimitOrder(req *api.LimitOrderRequest) (*api.OrderSubmissionResult, error) {
	var out api.OrderSubmissionResult
	err := c.doJSON(http.MethodPost, "/api/polymarket/orders/limit", req, &out)
	return &out, err
}

func (c *Client) CancelOrder(orderID string) error {
	u := c.base + "/api/polymarket/orders/" + url.PathEscape(orderID)
	r, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	c.addHeaders(r)
	resp, err := c.client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("executor: %s %s", resp.Status, string(b))
	}
	return nil
}

func (c *Client) GetOrder(orderID string) (json.RawMessage, error) {
	u := c.base + "/api/polymarket/orders/" + url.PathEscape(orderID)
	r, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(r)
	resp, err := c.client.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("executor: %s %s", resp.Status, string(b))
	}
	return json.RawMessage(b), nil
}

func (c *Client) GetBankroll() (*api.PolymarketBankrollResponse, error) {
	var out api.PolymarketBankrollResponse
	err := c.getJSON("/api/polymarket/bankroll", &out)
	return &out, err
}

func (c *Client) GetPositions(user string, limit, offset int) ([]api.PolymarketPosition, error) {
	q := url.Values{}
	if user != "" {
		q.Set("user", user)
	}
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	q.Set("limit", fmt.Sprintf("%d", limit))
	q.Set("offset", fmt.Sprintf("%d", offset))
	var out []api.PolymarketPosition
	err := c.getJSON("/api/polymarket/positions?"+q.Encode(), &out)
	return out, err
}

func (c *Client) addHeaders(r *http.Request) {
	r.Header.Set("Accept", "application/json")
	if c.liveAck {
		r.Header.Set(domain.HeaderLiveAck, "true")
	}
}

func (c *Client) getJSON(path string, out any) error {
	u := c.base + path
	r, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	c.addHeaders(r)
	resp, err := c.client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("executor GET %s: %s %s", path, resp.Status, truncate(string(b), 500))
	}
	return json.Unmarshal(b, out)
}

func (c *Client) doJSON(method, path string, body any, out any) error {
	u := c.base + path
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	r, err := http.NewRequest(method, u, &buf)
	if err != nil {
		return err
	}
	c.addHeaders(r)
	r.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("executor %s %s: %s %s", method, path, resp.Status, truncate(string(b), 500))
	}
	return json.Unmarshal(b, out)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
