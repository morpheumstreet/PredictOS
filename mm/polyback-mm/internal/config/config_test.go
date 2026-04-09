package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyLimitlessEnv(t *testing.T) {
	t.Setenv("LIMITLESS_API_KEY", "lk")
	t.Setenv("LIMITLESS_WALLET_ADDRESS", "0xw")
	l := LimitlessCfg{}
	applyLimitlessEnv(&l)
	if l.APIKey != "lk" || l.WalletAddress != "0xw" {
		t.Fatalf("%+v", l)
	}
	l2 := LimitlessCfg{APIKey: "yaml"}
	applyLimitlessEnv(&l2)
	if l2.APIKey != "yaml" || l2.WalletAddress != "0xw" {
		t.Fatal("yaml api_key should win; empty wallet from env")
	}
}

func TestApplyPredictFunEnv(t *testing.T) {
	t.Setenv("PREDICT_FUN_BASE_URL", "https://pf.example")
	t.Setenv("PREDICT_FUN_API_KEY", "pk")
	t.Setenv("PREDICT_FUN_PRIVATE_KEY", "0xsec")
	p := PredictFunCfg{}
	applyPredictFunEnv(&p)
	if p.BaseURL != "https://pf.example" || p.APIKey != "pk" || p.PrivateKey != "0xsec" {
		t.Fatalf("%+v", p)
	}
	p2 := PredictFunCfg{APIKey: "yaml"}
	applyPredictFunEnv(&p2)
	if p2.APIKey != "yaml" || p2.PrivateKey != "0xsec" {
		t.Fatal("empty fields should take env")
	}
}

func TestApplyKalshiDFlowEnv(t *testing.T) {
	t.Setenv("DFLOW_BASE_URL", "https://dflow.example")
	t.Setenv("DFLOW_API_KEY", "dk")
	t.Setenv("DFLOW_LIVE_EVENT_TICKER", "KX-1")
	k := KalshiDFlowCfg{}
	applyKalshiDFlowEnv(&k)
	if k.BaseURL != "https://dflow.example" || k.APIKey != "dk" || k.EventTicker != "KX-1" {
		t.Fatalf("%+v", k)
	}
}

func TestApplyIntelligenceEnv(t *testing.T) {
	t.Setenv("DOME_BASE_URL", "https://dome.example/v1")
	t.Setenv("DOME_API_KEY", "domek")
	t.Setenv("POLYFACTUAL_BASE_URL", "https://pf.example")
	t.Setenv("POLYFACTUAL_API_KEY", "pfk")
	i := IntelligenceCfg{}
	applyIntelligenceEnv(&i)
	if i.Dome.BaseURL != "https://dome.example/v1" || i.Dome.APIKey != "domek" {
		t.Fatalf("dome: %+v", i.Dome)
	}
	if i.Polyfactual.BaseURL != "https://pf.example" || i.Polyfactual.APIKey != "pfk" {
		t.Fatalf("polyfactual: %+v", i.Polyfactual)
	}
	i2 := IntelligenceCfg{Dome: DomeAPICfg{APIKey: "yaml-dome"}}
	applyIntelligenceEnv(&i2)
	if i2.Dome.APIKey != "yaml-dome" || i2.Dome.BaseURL != "https://dome.example/v1" {
		t.Fatal("yaml api_key should win; base_url still from env when empty in yaml")
	}
}

func TestApplyPolymarketEnv(t *testing.T) {
	t.Setenv("POLY_RELAYER_API_KEY", "relayer")
	t.Setenv("POLY_BUILDER_API_KEY", "bkey")
	t.Setenv("POLY_BUILDER_SECRET", "bsec")
	t.Setenv("POLY_BUILDER_PASSPHRASE", "bpass")
	p := PolymarketCfg{}
	applyPolymarketEnv(&p)
	if p.RelayerAPIKey != "relayer" || p.BuilderAPIKey != "bkey" || p.BuilderSecret != "bsec" || p.BuilderPassphrase != "bpass" {
		t.Fatalf("%+v", p)
	}
	p2 := PolymarketCfg{RelayerAPIKey: "yaml"}
	applyPolymarketEnv(&p2)
	if p2.RelayerAPIKey != "yaml" {
		t.Fatal("yaml should not be overwritten by env when set")
	}
	if p2.BuilderAPIKey != "bkey" {
		t.Fatal("empty yaml field should still take env")
	}

	t.Setenv("POLYGON_RPC_URLS", "https://rpc.a.example, https://rpc.b.example ")
	p3 := PolymarketCfg{}
	applyPolymarketEnv(&p3)
	if len(p3.PolygonRPCURLs) != 2 || p3.PolygonRPCURLs[0] != "https://rpc.a.example" {
		t.Fatalf("polygon rpcs: %v", p3.PolygonRPCURLs)
	}
	t.Setenv("POLYGON_RPC_URLS", "")
	t.Setenv("POLYGON_RPC_URL", "https://single.example")
	p4 := PolymarketCfg{}
	applyPolymarketEnv(&p4)
	if len(p4.PolygonRPCURLs) != 1 || p4.PolygonRPCURLs[0] != "https://single.example" {
		t.Fatalf("POLYGON_RPC_URL: %v", p4.PolygonRPCURLs)
	}
	p5 := PolymarketCfg{PolygonRPCURLs: []string{"https://yaml"}}
	t.Setenv("POLYGON_RPC_URL", "https://ignored.example")
	applyPolymarketEnv(&p5)
	if len(p5.PolygonRPCURLs) != 1 || p5.PolygonRPCURLs[0] != "https://yaml" {
		t.Fatal("yaml polygon list should block POLYGON_RPC_URL")
	}

	t.Setenv("POLYGON_RPC_CHAINLIST_URL", "https://chainlist.override.example/rpcs.json")
	p6 := PolymarketCfg{}
	applyPolymarketEnv(&p6)
	if p6.PolygonRPCChainlist.URL != "https://chainlist.override.example/rpcs.json" {
		t.Fatalf("chainlist url: %q", p6.PolygonRPCChainlist.URL)
	}
	t.Setenv("POLYGON_RPC_CHAINLIST_URL", "")
	t.Setenv("POLYGON_RPC_CHAINLIST_DISABLE", "true")
	p7 := PolymarketCfg{PolygonRPCChainlist: ChainlistPolygonRPCIngestCfg{Enabled: true}}
	applyPolymarketEnv(&p7)
	if p7.PolygonRPCChainlist.Enabled {
		t.Fatal("expected disabled by env")
	}
}

func TestLoad_mergesRealYML(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "develop.yaml")
	real := filepath.Join(dir, "real.yml")
	if err := os.WriteFile(base, []byte(`hft:
  mode: PAPER
  polymarket:
    gamma_url: https://gamma.example
    auth:
      private_key: ""
  limitless:
    api_key: ""
    wallet_address: ""
  predict_fun:
    api_key: ""
  kalshi_dflow:
    api_key: ""
kafka:
  brokers:
    - 127.0.0.1:9092
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(real, []byte(`hft:
  limitless:
    api_key: secret-from-real
    wallet_address: "0xabc"
`), 0644); err != nil {
		t.Fatal(err)
	}

	r, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if r.Hft.Mode != "PAPER" {
		t.Fatalf("mode: %q", r.Hft.Mode)
	}
	if r.Hft.Polymarket.GammaURL != "https://gamma.example" {
		t.Fatalf("gamma: %q", r.Hft.Polymarket.GammaURL)
	}
	if r.Hft.Limitless.APIKey != "secret-from-real" {
		t.Fatalf("limitless key: %q", r.Hft.Limitless.APIKey)
	}
	if r.Hft.Limitless.WalletAddress != "0xabc" {
		t.Fatalf("wallet: %q", r.Hft.Limitless.WalletAddress)
	}
	if len(r.Kafka.Brokers) != 1 || r.Kafka.Brokers[0] != "127.0.0.1:9092" {
		t.Fatalf("brokers: %v", r.Kafka.Brokers)
	}
	if len(r.Hft.Polymarket.PolygonRPCURLs) != len(DefaultPolygonRPCURLs) {
		t.Fatalf("polygon rpc defaults: got %d want %d", len(r.Hft.Polymarket.PolygonRPCURLs), len(DefaultPolygonRPCURLs))
	}
}

func TestLoad_polygonChainlistIngest(t *testing.T) {
	const body = `[{"chainId":137,"rpc":[{"url":"https://ingested-a.example"},{"url":"https://ingested-b.example"}]}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	dir := t.TempDir()
	base := filepath.Join(dir, "develop.yaml")
	yml := fmt.Sprintf(`hft:
  mode: PAPER
  polymarket:
    gamma_url: https://gamma.example
    chain_id: 137
    polygon_rpc_chainlist:
      enabled: true
      url: %q
      max_urls: 10
      timeout_seconds: 5
    auth:
      private_key: ""
  limitless:
    api_key: ""
  predict_fun:
    api_key: ""
  kalshi_dflow:
    api_key: ""
kafka:
  brokers:
    - 127.0.0.1:9092
`, srv.URL)
	if err := os.WriteFile(base, []byte(yml), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Hft.Polymarket.PolygonRPCURLs) != 2 {
		t.Fatalf("polygon rpc: %v", r.Hft.Polymarket.PolygonRPCURLs)
	}
}

func TestLoad_realTestingThenRealYML_order(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "develop.yaml")
	if err := os.WriteFile(base, []byte(`hft:
  mode: PAPER
  limitless:
    api_key: ""
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real.testing.yml"), []byte(`hft:
  limitless:
    api_key: from-testing
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "real.yml"), []byte(`hft:
  limitless:
    api_key: from-local
`), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if r.Hft.Limitless.APIKey != "from-local" {
		t.Fatalf("want real.yml to win, got %q", r.Hft.Limitless.APIKey)
	}
}

func TestLoad_fillsHftExecutorFromPublicAPI(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "develop.yaml")
	if err := os.WriteFile(base, []byte(`hft:
  mode: PAPER
  executor:
    base_url: ""
  polymarket:
    gamma_url: https://gamma.example
    auth:
      private_key: ""
  limitless:
    api_key: ""
  predict_fun:
    api_key: ""
  kalshi_dflow:
    api_key: ""
server:
  public_api_base_url: http://poly.example:7777
kafka:
  brokers:
    - 127.0.0.1:9092
`), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if r.Hft.Executor.BaseURL != "http://poly.example:7777" {
		t.Fatalf("want executor base_url from public_api_base_url, got %q", r.Hft.Executor.BaseURL)
	}
}

func TestLoad_appliesDefaultAPIBaseURLs(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "develop.yaml")
	if err := os.WriteFile(base, []byte(`hft:
  mode: PAPER
  polymarket:
    gamma_url: ""
    clob_rest_url: "  "
    clob_ws_url: ""
    auth:
      private_key: ""
  limitless:
    base_url: ""
    api_key: ""
  predict_fun:
    base_url: ""
    api_key: ""
  kalshi_dflow:
    base_url: ""
    api_key: ""
intelligence:
  dome:
    base_url: ""
    api_key: ""
  polyfactual:
    base_url: ""
    api_key: ""
ingestor:
  polymarket:
    data_api_base_url: ""
kafka:
  brokers:
    - 127.0.0.1:9092
`), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if r.Intelligence.Dome.BaseURL != DefaultDomeAPIBaseURL {
		t.Fatalf("dome base: %q", r.Intelligence.Dome.BaseURL)
	}
	if r.Intelligence.Polyfactual.BaseURL != DefaultPolyfactualAPIBaseURL {
		t.Fatalf("polyfactual base: %q", r.Intelligence.Polyfactual.BaseURL)
	}
	if r.Hft.KalshiDFlow.BaseURL != DefaultDFlowAPIBaseURL {
		t.Fatalf("dflow base: %q", r.Hft.KalshiDFlow.BaseURL)
	}
	if r.Hft.PredictFun.BaseURL != DefaultPredictFunAPIBaseURL {
		t.Fatalf("predict_fun base: %q", r.Hft.PredictFun.BaseURL)
	}
	if r.Hft.Limitless.BaseURL != DefaultLimitlessAPIBaseURL {
		t.Fatalf("limitless base: %q", r.Hft.Limitless.BaseURL)
	}
	if r.Hft.Polymarket.GammaURL != DefaultPolymarketGammaURL {
		t.Fatalf("gamma: %q", r.Hft.Polymarket.GammaURL)
	}
	if r.Hft.Polymarket.ClobRestURL != DefaultPolymarketClobRestURL {
		t.Fatalf("clob rest: %q", r.Hft.Polymarket.ClobRestURL)
	}
	if r.Hft.Polymarket.ClobWsURL != DefaultPolymarketClobWsURL {
		t.Fatalf("clob ws: %q", r.Hft.Polymarket.ClobWsURL)
	}
	if r.Ingestor.Polymarket.DataAPIBaseURL != DefaultPolymarketDataAPIBaseURL {
		t.Fatalf("data api: %q", r.Ingestor.Polymarket.DataAPIBaseURL)
	}
}

func TestLoad_missingRealYML_ok(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "develop.yaml")
	if err := os.WriteFile(base, []byte(`hft:
  mode: PAPER
kafka:
  brokers:
    - a:1
`), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if r.Hft.Mode != "PAPER" {
		t.Fatal(r.Hft.Mode)
	}
}
