package polymarket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms"
	"github.com/shopspring/decimal"
)

func TestPolymarket_GetAllMarkets_GetMarket_GetOrderbook_httptest(t *testing.T) {
	gammaBody := []map[string]interface{}{{
		"id":           "gm1",
		"slug":         "test-slug",
		"question":     "Will it rain?",
		"outcomes":     []map[string]int{{"price": 55}, {"price": 45}},
		"clobTokenIds": []string{"yesTok", "noTok"},
		"volume24hr":   1000.0,
		"liquidity":    500.0,
		"active":       true,
		"closed":       false,
	}}
	gammaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("gamma: method %s", r.Method)
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		switch {
		case r.URL.Path == "/markets" && r.URL.Query().Get("active") == "true":
			_ = json.NewEncoder(w).Encode(gammaBody)
		case r.URL.Path == "/markets/gm1":
			_ = json.NewEncoder(w).Encode(gammaBody[0])
		default:
			http.NotFound(w, r)
		}
	}))
	defer gammaSrv.Close()

	clobSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/book" || r.URL.Query().Get("token_id") != "yesTok" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"bids":[{"price":"0.54","size":"10"}],"asks":[{"price":"0.56","size":"20"}]}`))
	}))
	defer clobSrv.Close()

	p := NewPolymarket(gammaSrv.URL, clobSrv.URL)
	ctx := context.Background()

	if p.Name() != "polymarket" {
		t.Fatalf("name %q", p.Name())
	}

	markets, err := p.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(markets) != 1 || markets[0].ID != "gm1" {
		t.Fatalf("markets %+v", markets)
	}
	if !markets[0].YesPrice.Equal(decimal.RequireFromString("0.55")) {
		t.Fatalf("yes %s", markets[0].YesPrice)
	}

	m, err := p.GetMarket(ctx, "gm1")
	if err != nil {
		t.Fatal(err)
	}
	if m.YesTokenID != "yesTok" {
		t.Fatalf("token %q", m.YesTokenID)
	}

	yes, no, err := p.GetPrices(ctx, "gm1")
	if err != nil {
		t.Fatal(err)
	}
	if !yes.Equal(decimal.RequireFromString("0.55")) || !no.Equal(decimal.RequireFromString("0.45")) {
		t.Fatalf("prices yes=%s no=%s", yes, no)
	}

	ob, err := p.GetOrderbook(ctx, "gm1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ob.Bids) != 1 || len(ob.Asks) != 1 {
		t.Fatalf("book %+v", ob)
	}
	if !ob.Bids[0].Price.Equal(decimal.RequireFromString("0.54")) {
		t.Fatalf("bid price %s", ob.Bids[0].Price)
	}

	if err := p.HealthCheck(ctx); err != nil {
		t.Fatal(err)
	}

	_, err = p.SendOrder(ctx, platforms.PlaceOrderRequest{})
	if err != platforms.ErrTradingNotImplemented {
		t.Fatalf("SendOrder: want ErrTradingNotImplemented, got %v", err)
	}
}
