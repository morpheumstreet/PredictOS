//go:build live

// Run against production HTTP APIs (network required):
//
//	go test -tags=live ./internal/platforms/... -count=1 -timeout 120s -v
//
// Optional env (skip subtests when unset):
//   PREDICT_FUN_API_KEY, PREDICT_FUN_PRIVATE_KEY — Predict.fun v1 API + JWT auth smoke
//   DFLOW_API_KEY, DFLOW_LIVE_EVENT_TICKER — Kalshi data via DFlow (e.g. KXVIX-25)
package platforms_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/kalshidflow"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/limitless"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/predictfun"
)

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

	c := limitless.NewLimitless("", "", "")
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
	key := os.Getenv("PREDICT_FUN_API_KEY")
	if key == "" {
		t.Skip("set PREDICT_FUN_API_KEY for live Predict.fun")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := predictfun.NewPredictFun("", key, "")
	ms, err := c.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatal("predict.fun: expected at least one market")
	}
	t.Logf("predict.fun: markets=%d first_id=%s", len(ms), ms[0].ID)

	pk := os.Getenv("PREDICT_FUN_PRIVATE_KEY")
	if pk == "" {
		t.Log("PREDICT_FUN_PRIVATE_KEY unset; skipping Authenticate smoke")
		return
	}
	c2 := predictfun.NewPredictFun("", key, pk)
	if err := c2.Authenticate(ctx); err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	bal, err := c2.GetBalance(ctx)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}
	t.Logf("predict.fun: authenticated balance keys=%d", len(bal))
}

func TestLive_KalshiDFlow_EventMarkets(t *testing.T) {
	apiKey := os.Getenv("DFLOW_API_KEY")
	event := os.Getenv("DFLOW_LIVE_EVENT_TICKER")
	if apiKey == "" || event == "" {
		t.Skip("set DFLOW_API_KEY and DFLOW_LIVE_EVENT_TICKER for live DFlow")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	k := kalshidflow.NewKalshiDFlow("", apiKey, event)
	ms, err := k.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatalf("dflow: no markets for event %q", event)
	}
	t.Logf("dflow: event=%s markets=%d first_ticker=%s", event, len(ms), ms[0].ID)
}
