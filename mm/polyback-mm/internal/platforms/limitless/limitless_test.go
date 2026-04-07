package limitless

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms"
	"github.com/shopspring/decimal"
)

func TestLimitless_httptest(t *testing.T) {
	mkt := map[string]interface{}{
		"id":          "L1",
		"slug":        "sl",
		"title":       "Limitless Q?",
		"prices":      []int{40, 60},
		"status":      "active",
		"volume":      42.5,
		"liquidity":   10.0,
		"positionIds": []string{"pYes", "pNo"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/markets/active":
			_ = json.NewEncoder(w).Encode([]interface{}{mkt})
		case r.Method == http.MethodGet && r.URL.Path == "/markets/L1":
			_ = json.NewEncoder(w).Encode(mkt)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/orderbook"):
			_, _ = w.Write([]byte(`{"bids":[{"price":"0.39","size":"5"}],"asks":[{"price":"0.41","size":"7"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/orders":
			if r.URL.Query().Get("walletAddress") == "" {
				http.Error(w, "wallet", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"orders": []map[string]interface{}{{
					"id": "o1", "marketId": "L1", "side": "YES", "size": "1", "price": "0.4", "type": "limit", "status": "open",
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/account/positions":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"positions": []map[string]interface{}{{
					"marketId": "L1", "side": "yes", "size": "2", "avgPrice": "0.4", "currentPrice": "0.41", "pnl": "0.1",
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/account/balance":
			_ = json.NewEncoder(w).Encode(map[string]string{"usdcBalance": "123.45", "points": "0"})
		case r.Method == http.MethodPost && r.URL.Path == "/orders":
			b, _ := io.ReadAll(r.Body)
			var body map[string]string
			_ = json.Unmarshal(b, &body)
			if body["walletAddress"] == "" {
				http.Error(w, "wallet", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"orderId": "new1", "status": "open"})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/orders/"):
			b, _ := io.ReadAll(r.Body)
			var body map[string]string
			_ = json.Unmarshal(b, &body)
			if body["walletAddress"] == "" {
				http.Error(w, "wallet", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewLimitless(srv.URL, "test-key", "0xabc")
	ctx := context.Background()

	if c.Name() != "limitless" {
		t.Fatalf("name %q", c.Name())
	}

	ms, err := c.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].ID != "L1" {
		t.Fatalf("markets %+v", ms)
	}
	if !ms[0].YesPrice.Equal(decimal.RequireFromString("0.4")) {
		t.Fatalf("yes %s", ms[0].YesPrice)
	}

	m, err := c.GetMarket(ctx, "L1")
	if err != nil {
		t.Fatal(err)
	}
	if m.Question != "Limitless Q?" {
		t.Fatalf("q %q", m.Question)
	}

	ob, err := c.GetOrderbook(ctx, "L1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ob.Bids) != 1 {
		t.Fatalf("bids %+v", ob.Bids)
	}

	yes, no, err := c.GetPrices(ctx, "L1")
	if err != nil {
		t.Fatal(err)
	}
	if !yes.Equal(decimal.RequireFromString("0.4")) {
		t.Fatalf("yes %s", yes)
	}
	_ = no

	orders, err := c.ListOrders(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 || orders[0].ID != "o1" {
		t.Fatalf("orders %+v", orders)
	}

	pos, err := c.GetPositions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pos) != 1 || pos[0].MarketID != "L1" {
		t.Fatalf("pos %+v", pos)
	}

	bal, err := c.GetBalance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bal["USDC"].Equal(decimal.RequireFromString("123.45")) {
		t.Fatalf("bal %+v", bal)
	}

	res, err := c.SendOrder(ctx, platforms.PlaceOrderRequest{MarketID: "L1", Side: "yes", Size: decimal.NewFromInt(1), Price: decimal.RequireFromString("0.4")})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success || res.OrderID != "new1" {
		t.Fatalf("send %+v", res)
	}

	if err := c.CancelOrder(ctx, "x"); err != nil {
		t.Fatal(err)
	}

	if err := c.HealthCheck(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestLimitless_ErrNotConfigured(t *testing.T) {
	c := NewLimitless("", "", "")
	ctx := context.Background()
	if _, err := c.GetAllMarkets(ctx); err != platforms.ErrNotConfigured {
		t.Fatalf("got %v", err)
	}
}
