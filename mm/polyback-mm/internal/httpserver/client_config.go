package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
)

// ClientConfigResponse is safe for browsers: no secrets, keys, or passwords.
type ClientConfigResponse struct {
	APIBaseURL   string                 `json:"apiBaseUrl"`
	HftMode      string                 `json:"hftMode"`
	Server       ClientConfigServerInfo `json:"server"`
	ServiceURLs  ClientServiceURLs      `json:"serviceUrls"`
	Modules      []ClientConfigModule   `json:"modules"`
}

// ClientServiceURLs maps each polyback process to an HTTP base URL for terminal relay (dev defaults :port → http://127.0.0.1:port).
type ClientServiceURLs struct {
	Executor         string `json:"executor,omitempty"`
	Strategy         string `json:"strategy,omitempty"`
	Analytics        string `json:"analytics,omitempty"`
	Ingestor         string `json:"ingestor,omitempty"`
	Infrastructure   string `json:"infrastructure,omitempty"`
}

// ClientConfigServerInfo exposes listen addresses and public URL (non-secret).
type ClientConfigServerInfo struct {
	PublicAPIBaseURL   string `json:"publicApiBaseUrl"`
	ExecutorAddr       string `json:"executorAddr"`
	StrategyAddr       string `json:"strategyAddr"`
	AnalyticsAddr      string `json:"analyticsAddr"`
	IngestorAddr       string `json:"ingestorAddr"`
	InfrastructureAddr string `json:"infrastructureAddr"`
}

// ClientConfigModule documents HTTP route groups for the terminal / ops UI.
type ClientConfigModule struct {
	Name       string `json:"name"`
	PathPrefix string `json:"pathPrefix"`
}

func clientConfigEnabled(s *config.ServerCfg) bool {
	if s.ClientConfigEnabled == nil {
		return true
	}
	return *s.ClientConfigEnabled
}

// ResolveAPIBaseURL returns server.public_api_base_url or a best-effort default from executor listen addr.
func ResolveAPIBaseURL(root *config.Root) string {
	u := strings.TrimSpace(root.Server.PublicAPIBaseURL)
	if u != "" {
		return strings.TrimSuffix(u, "/")
	}
	addr := strings.TrimSpace(root.Server.ExecutorAddr)
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	if addr != "" && !strings.Contains(addr, "://") {
		return "http://" + addr
	}
	return ""
}

// ListenAddrToBaseURL converts a listen address like ":8080" or "127.0.0.1:8080" into a browser-reachable base URL.
func ListenAddrToBaseURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if strings.Contains(addr, "://") {
		return strings.TrimSuffix(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	return "http://" + addr
}

// BuildClientConfigResponse builds the JSON payload for GET /api/v1/config/client.
func BuildClientConfigResponse(root *config.Root) ClientConfigResponse {
	return ClientConfigResponse{
		APIBaseURL: ResolveAPIBaseURL(root),
		HftMode:    strings.TrimSpace(root.Hft.Mode),
		Server: ClientConfigServerInfo{
			PublicAPIBaseURL:   strings.TrimSpace(root.Server.PublicAPIBaseURL),
			ExecutorAddr:       root.Server.ExecutorAddr,
			StrategyAddr:       root.Server.StrategyAddr,
			AnalyticsAddr:      root.Server.AnalyticsAddr,
			IngestorAddr:       root.Server.IngestorAddr,
			InfrastructureAddr: root.Server.InfrastructureAddr,
		},
		ServiceURLs: ClientServiceURLs{
			Executor:       ListenAddrToBaseURL(root.Server.ExecutorAddr),
			Strategy:       ListenAddrToBaseURL(root.Server.StrategyAddr),
			Analytics:      ListenAddrToBaseURL(root.Server.AnalyticsAddr),
			Ingestor:       ListenAddrToBaseURL(root.Server.IngestorAddr),
			Infrastructure: ListenAddrToBaseURL(root.Server.InfrastructureAddr),
		},
		Modules: []ClientConfigModule{
			{Name: "polymarket_executor", PathPrefix: "/api/polymarket"},
			{Name: "strategy", PathPrefix: "/api/strategy"},
			{Name: "ingestor", PathPrefix: "/api/ingestor"},
			{Name: "analytics", PathPrefix: "/api/analytics"},
			{Name: "infrastructure", PathPrefix: "/api/infrastructure"},
		},
	}
}

// MountClientConfig registers GET /api/v1/config/client when enabled in config.
func MountClientConfig(r chi.Router, root *config.Root) {
	if root == nil || !clientConfigEnabled(&root.Server) {
		return
	}
	r.Get("/api/v1/config/client", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(BuildClientConfigResponse(root))
	})
}
