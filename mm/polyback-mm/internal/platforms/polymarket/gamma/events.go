package gamma

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// EventBySlugPath fetches GET /events/{slug} (path form; some slugs work only here).
func (c *Client) EventBySlugPath(slug string) (json.RawMessage, error) {
	u := c.baseURL + "/events/" + url.PathEscape(slug)
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
		return nil, fmt.Errorf("gamma /events/{slug}: status %d body=%s", resp.StatusCode, string(b))
	}
	return json.RawMessage(b), nil
}
