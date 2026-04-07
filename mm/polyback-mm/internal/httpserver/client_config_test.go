package httpserver

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
)

func TestBuildClientConfigResponse_NoSecretLeak(t *testing.T) {
	root := &config.Root{
		Hft: config.Hft{
			Mode: "PAPER",
			Polymarket: config.PolymarketCfg{
				RelayerAPIKey: "SECRET_RELAYER_SHOULD_NOT_APPEAR",
				BuilderAPIKey: "SECRET_BUILDER",
				Auth:          config.AuthCfg{PrivateKey: "0xdeadbeef"},
			},
		},
		Server: config.ServerCfg{
			PublicAPIBaseURL: "http://127.0.0.1:8080",
			ExecutorAddr:     ":8080",
		},
	}
	b, err := json.Marshal(BuildClientConfigResponse(root))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, forbidden := range []string{"SECRET", "0xdead", "relayer", "builder", "private_key"} {
		if strings.Contains(strings.ToLower(s), strings.ToLower(forbidden)) {
			t.Fatalf("response may leak secrets (matched %q): %s", forbidden, s)
		}
	}
}

func TestListenAddrToBaseURL(t *testing.T) {
	if got := ListenAddrToBaseURL(":9091"); got != "http://127.0.0.1:9091" {
		t.Fatalf("got %q", got)
	}
	if got := ListenAddrToBaseURL("192.168.1.2:4000"); got != "http://192.168.1.2:4000" {
		t.Fatalf("got %q", got)
	}
	if got := ListenAddrToBaseURL(""); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAPIBaseURL(t *testing.T) {
	r := &config.Root{
		Server: config.ServerCfg{PublicAPIBaseURL: "https://x.com/poly/"},
	}
	if got := ResolveAPIBaseURL(r); got != "https://x.com/poly" {
		t.Fatalf("got %q", got)
	}
	r2 := &config.Root{Server: config.ServerCfg{ExecutorAddr: ":9090"}}
	if got := ResolveAPIBaseURL(r2); got != "http://127.0.0.1:9090" {
		t.Fatalf("got %q", got)
	}
}
