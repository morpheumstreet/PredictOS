package gamma

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// MarketBySlug fetches GET /markets/slug/{slug}.
func (c *Client) MarketBySlug(slug string) ([]byte, error) {
	u := c.baseURL + "/markets/slug/" + url.PathEscape(slug)
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
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("404 market not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gamma /markets/slug: status %d body=%s", resp.StatusCode, string(b))
	}
	return b, nil
}
