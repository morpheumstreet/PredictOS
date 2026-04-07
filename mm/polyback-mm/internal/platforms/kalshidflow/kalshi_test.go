package kalshidflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms"
	"github.com/shopspring/decimal"
)

func TestKalshiDFlow_httptest(t *testing.T) {
	payload := map[string]interface{}{
		"event_ticker": "KXTEST",
		"markets": []map[string]interface{}{{
			"ticker":       "KXTEST-YES",
			"event_ticker": "KXTEST",
			"title":        "Will X happen?",
			"subtitle":     "sub",
			"status":       "open",
			"close_time":   "2026-12-31T23:59:59Z",
			"yes_bid":      48.0,
			"yes_ask":      52.0,
			"no_bid":       48.0,
			"no_ask":       52.0,
			"volume_24h":   1000.0,
			"liquidity":    200.0,
		}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "dflow-key" {
			http.Error(w, "key", http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/event/KXTEST" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("withNestedMarkets") != "true" {
			http.Error(w, "param", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	k := NewKalshiDFlow(srv.URL, "dflow-key", "KXTEST")
	ctx := context.Background()

	if k.Name() != "kalshi_dflow" {
		t.Fatalf("name %q", k.Name())
	}

	ms, err := k.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].ID != "KXTEST-YES" {
		t.Fatalf("markets %+v", ms)
	}
	// mid yes (48+52)/2 / 100 = 0.5
	if !ms[0].YesPrice.Equal(decimal.RequireFromString("0.5")) {
		t.Fatalf("yes %s", ms[0].YesPrice)
	}

	m, err := k.GetMarket(ctx, "KXTEST-YES")
	if err != nil {
		t.Fatal(err)
	}
	if m.Question != "Will X happen?" {
		t.Fatalf("q %q", m.Question)
	}

	yes, no, err := k.GetPrices(ctx, "KXTEST-YES")
	if err != nil {
		t.Fatal(err)
	}
	if !yes.Equal(decimal.RequireFromString("0.5")) || !no.Equal(decimal.RequireFromString("0.5")) {
		t.Fatalf("yes=%s no=%s", yes, no)
	}

	if err := k.HealthCheck(ctx); err != nil {
		t.Fatal(err)
	}

	_, err = k.GetOrderbook(ctx, "x")
	if err == nil {
		t.Fatal("expected orderbook error")
	}

	_, err = k.SendOrder(ctx, platforms.PlaceOrderRequest{})
	if err != platforms.ErrTradingNotImplemented {
		t.Fatalf("SendOrder: %v", err)
	}
}

func TestKalshiDFlow_ErrNotConfigured(t *testing.T) {
	k := NewKalshiDFlow("", "", "")
	ctx := context.Background()
	if _, err := k.GetAllMarkets(ctx); err != platforms.ErrNotConfigured {
		t.Fatalf("got %v", err)
	}
}
