package config

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/evmrpc"
)

// DefaultChainlistRPCJSONURL is the published ChainList rpcs bundle (format: array of { chainId, rpc }).
const DefaultChainlistRPCJSONURL = "https://chainlist.org/rpcs.json"

// NewPolygonEVMRPCManager returns an evmrpc.Manager backed by Polymarket polygon_rpc_urls after config.Load
// (chainlist ingest and static defaults already applied). refreshEvery <= 0 uses the Manager default (5m).
func NewPolygonEVMRPCManager(r *Root, refreshEvery time.Duration) *evmrpc.Manager {
	var u []string
	if r != nil {
		u = r.Hft.Polymarket.PolygonRPCURLs
	}
	return evmrpc.NewManager(u, refreshEvery)
}

func applyPolygonRPCChainlistIngest(r *Root) {
	if r == nil || polygonRPCURLsExplicitlySet(&r.Hft.Polymarket) {
		return
	}
	p := &r.Hft.Polymarket
	if !p.PolygonRPCChainlist.Enabled {
		return
	}
	feed := strings.TrimSpace(p.PolygonRPCChainlist.URL)
	if feed == "" {
		feed = DefaultChainlistRPCJSONURL
	}
	timeout := chainlistFetchTimeout(p.PolygonRPCChainlist.TimeoutSeconds)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	hc := &http.Client{Timeout: timeout}
	urls, err := evmrpc.FetchHTTPSRPCsForChain(ctx, hc, feed, effectivePolygonChainID(p.ChainID), p.PolygonRPCChainlist.MaxURLs)
	if err != nil || len(urls) == 0 {
		return
	}
	p.PolygonRPCURLs = urls
}

func polygonRPCURLsExplicitlySet(p *PolymarketCfg) bool {
	return len(nonEmptyTrimmedStrings(p.PolygonRPCURLs)) > 0
}

func effectivePolygonChainID(yamlChainID int) int64 {
	if yamlChainID == 0 {
		return int64(DefaultPolygonChainID)
	}
	return int64(yamlChainID)
}

func chainlistFetchTimeout(timeoutSeconds int) time.Duration {
	if timeoutSeconds <= 0 {
		timeoutSeconds = DefaultPolygonRPCChainlistTimeoutSeconds
	}
	return time.Duration(timeoutSeconds) * time.Second
}
