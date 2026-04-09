package evmrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultChainlistHTTPSCap is the maximum HTTPS RPC URLs kept when maxN is unset or non-positive.
const DefaultChainlistHTTPSCap = 64

// chainlistNetwork matches entries in https://chainlist.org/rpcs.json (array of networks).
type chainlistNetwork struct {
	ChainID float64 `json:"chainId"`
	RPC     []struct {
		URL string `json:"url"`
	} `json:"rpc"`
}

// FetchHTTPSRPCsForChain downloads feedURL (ChainList rpcs.json format), finds the network with chainID,
// and returns up to maxN deduplicated https:// RPC URLs (wss skipped — ethclient uses HTTP).
func FetchHTTPSRPCsForChain(ctx context.Context, hc *http.Client, feedURL string, chainID int64, maxN int) ([]string, error) {
	if hc == nil {
		hc = http.DefaultClient
	}
	if maxN <= 0 {
		maxN = DefaultChainlistHTTPSCap
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "PredictOS-polyback-mm/1.0")
	resp, err := hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("chainlist fetch: %s %s", resp.Status, strings.TrimSpace(string(b)))
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var networks []chainlistNetwork
	if err := json.Unmarshal(b, &networks); err != nil {
		return nil, fmt.Errorf("chainlist json: %w", err)
	}
	var candidates []string
	for _, n := range networks {
		if int64(n.ChainID) != chainID {
			continue
		}
		for _, e := range n.RPC {
			u := strings.TrimSpace(e.URL)
			if !strings.HasPrefix(strings.ToLower(u), "https://") {
				continue
			}
			candidates = append(candidates, u)
		}
		break
	}
	out := normalizeURLs(candidates)
	if len(out) > maxN {
		out = out[:maxN]
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("chainlist: no https rpc for chainId %d", chainID)
	}
	return out, nil
}
