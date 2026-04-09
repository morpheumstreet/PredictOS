package polyfactual

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultBaseURL = "https://deep-research-api.thekid-solana.workers.dev"
const maxQueryLen = 1000
const defaultTimeout = 5 * time.Minute

// Client calls the Polyfactual Deep Research HTTP API.
type Client struct {
	baseURL        string
	configuredKey  string
	http           *http.Client
}

// NewClient builds a client. Pass empty baseURL and apiKey to read only from POLYFACTUAL_BASE_URL / POLYFACTUAL_API_KEY (legacy). After config.Load, prefer passing root.Intelligence.Polyfactual values so YAML can supply credentials.
func NewClient(hc *http.Client, baseURL, apiKey string) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: defaultTimeout}
	}
	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = strings.TrimSpace(os.Getenv("POLYFACTUAL_BASE_URL"))
	}
	if base == "" {
		base = defaultBaseURL
	}
	return &Client{
		baseURL:       strings.TrimSuffix(base, "/"),
		configuredKey: strings.TrimSpace(apiKey),
		http:          hc,
	}
}

func (c *Client) resolveAPIKey() (string, error) {
	if k := strings.TrimSpace(c.configuredKey); k != "" {
		return k, nil
	}
	k := strings.TrimSpace(os.Getenv("POLYFACTUAL_API_KEY"))
	if k == "" {
		return "", fmt.Errorf("POLYFACTUAL_API_KEY is not set")
	}
	return k, nil
}

// AnswerRequest matches the edge function body.
type AnswerRequest struct {
	Query string `json:"query"`
	Text  *bool  `json:"text,omitempty"`
}

// GenerateAnswer POSTs /answer and returns the decoded success payload (mirrors TS PolyfactualResponse).
func (c *Client) GenerateAnswer(req AnswerRequest) (map[string]any, error) {
	key, err := c.resolveAPIKey()
	if err != nil {
		return nil, err
	}
	q := strings.TrimSpace(req.Query)
	if q == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if len(q) > maxQueryLen {
		return nil, fmt.Errorf("query exceeds maximum length of %d characters", maxQueryLen)
	}
	text := true
	if req.Text != nil {
		text = *req.Text
	}
	body, _ := json.Marshal(map[string]any{"query": q, "text": text})
	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/answer", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", key)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var wrap map[string]any
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, fmt.Errorf("polyfactual: invalid json: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("polyfactual API error: %d", resp.StatusCode)
		if e, ok := wrap["error"].(string); ok && e != "" {
			errMsg = e
		}
		return nil, fmt.Errorf("%s", errMsg)
	}
	if success, ok := wrap["success"].(bool); ok && !success {
		if e, ok := wrap["error"].(string); ok {
			return nil, fmt.Errorf("%s", e)
		}
		return nil, fmt.Errorf("polyfactual: unsuccessful response")
	}
	return wrap, nil
}

// Health GET /health
func (c *Client) Health() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
