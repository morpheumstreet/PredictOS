package platforms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func doJSON(ctx context.Context, client *http.Client, method, urlStr string, headers map[string]string, body []byte) ([]byte, int, error) {
	if client == nil {
		client = newHTTPClient()
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return b, resp.StatusCode, nil
}

func doJSONExpect2xx(ctx context.Context, client *http.Client, method, urlStr string, headers map[string]string, body []byte) ([]byte, error) {
	b, code, err := doJSON(ctx, client, method, urlStr, headers, body)
	if err != nil {
		return nil, err
	}
	if code < 200 || code >= 300 {
		return nil, fmt.Errorf("platforms http %s %s: status %d body=%s", method, urlStr, code, truncate(string(b), 512))
	}
	return b, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// DefaultHTTPClient returns an HTTP client with the standard timeout used by platform adapters.
func DefaultHTTPClient() *http.Client {
	return newHTTPClient()
}

// DoJSONExpect2xx performs a request and returns the response body when status is 2xx.
func DoJSONExpect2xx(ctx context.Context, client *http.Client, method, urlStr string, headers map[string]string, body []byte) ([]byte, error) {
	if client == nil {
		client = newHTTPClient()
	}
	return doJSONExpect2xx(ctx, client, method, urlStr, headers, body)
}

// DecodeJSONArrayEnvelope extracts a JSON array from a top-level array or {data:[]} / {markets:[]} envelopes.
func DecodeJSONArrayEnvelope(raw []byte) ([]json.RawMessage, error) {
	var direct []json.RawMessage
	if err := json.Unmarshal(raw, &direct); err == nil && len(direct) > 0 {
		return direct, nil
	}
	var env struct {
		Data    []json.RawMessage `json:"data"`
		Markets []json.RawMessage `json:"markets"`
	}
	if err := json.Unmarshal(raw, &env); err == nil {
		if len(env.Data) > 0 {
			return env.Data, nil
		}
		if len(env.Markets) > 0 {
			return env.Markets, nil
		}
	}
	return nil, fmt.Errorf("platforms: no array payload in response")
}
