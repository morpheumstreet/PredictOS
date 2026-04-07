//go:build live

// Run against production HTTP APIs (network required):
//
//	go test -tags=live ./internal/platforms/... -count=1 -timeout 120s -v
//
// Config: POLYBACK_CONFIG or defaults to ../../configs/develop.yaml (from this package dir),
// merged with real.testing.yml / real.yml and env fallbacks (see internal/config).
//
// Optional credentials (YAML hft.* or env — skip subtests when still empty after Load):
//
//	predict_fun: PREDICT_FUN_API_KEY, PREDICT_FUN_PRIVATE_KEY
//	kalshi_dflow: DFLOW_API_KEY, DFLOW_LIVE_EVENT_TICKER
package platforms_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/wiring"
)

func liveRoot(t *testing.T) *config.Root {
	t.Helper()
	path := os.Getenv("POLYBACK_CONFIG")
	if path == "" {
		path = filepath.Join("..", "..", "configs", "develop.yaml")
	}
	root, err := config.Load(path)
	if err != nil {
		t.Fatalf("config load %q: %v", path, err)
	}
	return root
}

func TestLive_Polymarket_GammaAndCLOB(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := polymarket.NewPolymarket("", "")
	ms, err := c.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatal("gamma: expected at least one market")
	}

	var marketID string
	for _, m := range ms {
		if m.YesTokenID != "" {
			marketID = m.ID
			break
		}
	}
	if marketID == "" {
		t.Fatal("gamma: expected clobTokenIds on at least one market (string or array)")
	}

	ob, err := c.GetOrderbook(ctx, marketID)
	if err != nil {
		t.Fatalf("clob book: %v", err)
	}
	t.Logf("polymarket: markets=%d sample_id=%s orderbook_bids=%d orderbook_asks=%d",
		len(ms), marketID, len(ob.Bids), len(ob.Asks))
}

func TestLive_Limitless_ActiveMarkets(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	root := liveRoot(t)
	c := wiring.LimitlessFromHft(&root.Hft)
	ms, err := c.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatal("limitless: expected active markets")
	}
	nActive := 0
	for _, m := range ms {
		if m.Active {
			nActive++
		}
	}
	if nActive == 0 {
		t.Fatal("limitless: expected at least one market parsed as active")
	}
	t.Logf("limitless: markets=%d active_parsed=%d first_id=%s", len(ms), nActive, ms[0].ID)
}

func TestLive_PredictFun_MarketsWithAPIKey(t *testing.T) {
	root := liveRoot(t)
	if strings.TrimSpace(root.Hft.PredictFun.APIKey) == "" {
		t.Skip("set PREDICT_FUN_API_KEY or hft.predict_fun.api_key for live Predict.fun")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := wiring.PredictFunFromHft(&root.Hft)
	ms, err := c.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatal("predict.fun: expected at least one market")
	}
	t.Logf("predict.fun: markets=%d first_id=%s", len(ms), ms[0].ID)

	if strings.TrimSpace(root.Hft.PredictFun.PrivateKey) == "" {
		t.Log("private key unset; skipping Authenticate smoke")
		return
	}
	if err := c.Authenticate(ctx); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	bal, err := c.GetBalance(ctx)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	t.Logf("predict.fun: authenticated balance keys=%d", len(bal))
}

func TestLive_KalshiDFlow_EventMarkets(t *testing.T) {
	root := liveRoot(t)
	apiKey := strings.TrimSpace(root.Hft.KalshiDFlow.APIKey)
	event := strings.TrimSpace(root.Hft.KalshiDFlow.EventTicker)
	if apiKey == "" || event == "" {
		t.Skip("set DFLOW_API_KEY + DFLOW_LIVE_EVENT_TICKER or hft.kalshi_dflow for live DFlow")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	k := wiring.KalshiDFlowFromHft(&root.Hft)
	ms, err := k.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatalf("dflow: no markets for event %q", event)
	}
	t.Logf("dflow: event=%s markets=%d first_ticker=%s", event, len(ms), ms[0].ID)
}
