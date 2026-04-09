package config

import "strings"

// Default API bases when YAML base_url / *_url is empty or whitespace-only (after merge + env fill).
// Mirrors hardcoded fallbacks in: internal/platforms/dome, polyfactual, kalshidflow, predictfun,
// limitless, polymarket (gamma/clob), and ingestor Polymarket data API. Update both when a vendor URL changes.
const (
	DefaultDomeAPIBaseURL           = "https://api.domeapi.io/v1"
	DefaultPolyfactualAPIBaseURL    = "https://deep-research-api.thekid-solana.workers.dev"
	DefaultDFlowAPIBaseURL          = "https://a.prediction-markets-api.dflow.net/api/v1"
	DefaultPredictFunAPIBaseURL     = "https://api.predict.fun/v1"
	DefaultLimitlessAPIBaseURL      = "https://api.limitless.exchange"
	DefaultPolymarketGammaURL       = "https://gamma-api.polymarket.com"
	DefaultPolymarketClobRestURL    = "https://clob.polymarket.com"
	DefaultPolymarketClobWsURL      = "wss://ws-subscriptions-clob.polymarket.com"
	DefaultPolymarketDataAPIBaseURL = "https://data-api.polymarket.com"
)

// Polygon mainnet (used for default JSON-RPC chain id and ChainList ingest when chain_id is unset).
const DefaultPolygonChainID = 137

// DefaultPolygonRPCChainlistTimeoutSeconds is the HTTP budget for ChainList rpcs.json fetch when timeout_seconds is unset.
const DefaultPolygonRPCChainlistTimeoutSeconds = 30

// DefaultPolygonRPCURLs are public Polygon (DefaultPolygonChainID) JSON-RPC HTTPS endpoints used when
// hft.polymarket.polygon_rpc_urls is empty. Curate from https://chainlist.org (Polygon Mainnet).
// Selection at runtime uses latency probing (see internal/evmrpc, morpheum-labs/pricefeeding rpcscan).
var DefaultPolygonRPCURLs = []string{
	"https://polygon-bor-rpc.publicnode.com",
	"https://1rpc.io/matic",
	"https://polygon.drpc.org",
	"https://polygon-rpc.com",
}

func applyDefaultAPIBaseURLs(r *Root) {
	if r == nil {
		return
	}
	if strings.TrimSpace(r.Intelligence.Dome.BaseURL) == "" {
		r.Intelligence.Dome.BaseURL = DefaultDomeAPIBaseURL
	}
	if strings.TrimSpace(r.Intelligence.Polyfactual.BaseURL) == "" {
		r.Intelligence.Polyfactual.BaseURL = DefaultPolyfactualAPIBaseURL
	}
	if strings.TrimSpace(r.Hft.KalshiDFlow.BaseURL) == "" {
		r.Hft.KalshiDFlow.BaseURL = DefaultDFlowAPIBaseURL
	}
	if strings.TrimSpace(r.Hft.PredictFun.BaseURL) == "" {
		r.Hft.PredictFun.BaseURL = DefaultPredictFunAPIBaseURL
	}
	if strings.TrimSpace(r.Hft.Limitless.BaseURL) == "" {
		r.Hft.Limitless.BaseURL = DefaultLimitlessAPIBaseURL
	}
	pm := &r.Hft.Polymarket
	if strings.TrimSpace(pm.GammaURL) == "" {
		pm.GammaURL = DefaultPolymarketGammaURL
	}
	if strings.TrimSpace(pm.ClobRestURL) == "" {
		pm.ClobRestURL = DefaultPolymarketClobRestURL
	}
	if strings.TrimSpace(pm.ClobWsURL) == "" {
		pm.ClobWsURL = DefaultPolymarketClobWsURL
	}
	if strings.TrimSpace(r.Ingestor.Polymarket.DataAPIBaseURL) == "" {
		r.Ingestor.Polymarket.DataAPIBaseURL = DefaultPolymarketDataAPIBaseURL
	}
}

func applyDefaultPolygonRPCs(p *PolymarketCfg) {
	if p == nil {
		return
	}
	filtered := nonEmptyTrimmedStrings(p.PolygonRPCURLs)
	if len(filtered) > 0 {
		p.PolygonRPCURLs = filtered
		return
	}
	p.PolygonRPCURLs = append([]string(nil), DefaultPolygonRPCURLs...)
}

func nonEmptyTrimmedStrings(in []string) []string {
	var out []string
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}
