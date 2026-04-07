package predictfun

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms"
	"github.com/shopspring/decimal"
)

func TestPredictFun_httptest(t *testing.T) {
	key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pkHex := fmt.Sprintf("%x", crypto.FromECDSA(key))
	addr := crypto.PubkeyToAddress(key.PublicKey).Hex()

	var jwtIssued string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/auth/message":
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "sign-me-predictfun-test"})
		case r.Method == http.MethodPost && r.URL.Path == "/auth/jwt":
			var body struct {
				Message, Signature, Address string
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Address != addr || body.Message != "sign-me-predictfun-test" || !strings.HasPrefix(body.Signature, "0x") {
				http.Error(w, "bad auth", http.StatusUnauthorized)
				return
			}
			jwtIssued = "test-jwt-token"
			_ = json.NewEncoder(w).Encode(map[string]string{"token": jwtIssued})
		case r.Method == http.MethodGet && r.URL.Path == "/markets":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"markets": []map[string]interface{}{{
					"id": "PF1", "question": "PF?", "yesPrice": 0.52, "noPrice": 0.48,
					"volume24h": 1.0, "liquidity": 2.0, "status": "active",
				}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/markets/PF1":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "PF1", "question": "PF?", "yesPrice": 0.52, "noPrice": 0.48,
				"volume24h": 1.0, "liquidity": 2.0, "status": "active",
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/orderbook"):
			_, _ = w.Write([]byte(`{"bids":[{"price":0.51,"size":3}],"asks":[{"price":0.53,"size":4}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/orders":
			auth := r.Header.Get("Authorization")
			if !strings.Contains(auth, jwtIssued) {
				http.Error(w, "jwt", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"orders": []map[string]interface{}{{"id": 99, "marketId": "PF1", "side": "YES", "size": "1", "price": "0.5", "type": "LIMIT", "status": "OPEN"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/positions":
			auth := r.Header.Get("Authorization")
			if !strings.Contains(auth, jwtIssued) {
				http.Error(w, "jwt", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"positions": []map[string]interface{}{{"marketId": "PF1", "side": "yes", "size": "1", "avgPrice": "0.5", "currentPrice": "0.51", "pnl": "0"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/account":
			auth := r.Header.Get("Authorization")
			if !strings.Contains(auth, jwtIssued) {
				http.Error(w, "jwt", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]float64{"balance": 50, "lockedBalance": 5})
		case r.Method == http.MethodPost && r.URL.Path == "/orders":
			auth := r.Header.Get("Authorization")
			if !strings.Contains(auth, jwtIssued) {
				http.Error(w, "jwt", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "ord1", "status": "OPEN"})
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/orders/"):
			auth := r.Header.Get("Authorization")
			if !strings.Contains(auth, jwtIssued) {
				http.Error(w, "jwt", http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewPredictFun(srv.URL, "pf-key", pkHex)
	ctx := context.Background()

	if c.Name() != "predict_fun" {
		t.Fatalf("name %q", c.Name())
	}

	ms, err := c.GetAllMarkets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].ID != "PF1" {
		t.Fatalf("markets %+v", ms)
	}

	m, err := c.GetMarket(ctx, "PF1")
	if err != nil {
		t.Fatal(err)
	}
	if !m.YesPrice.Equal(decimal.RequireFromString("0.52")) {
		t.Fatalf("yes %s", m.YesPrice)
	}

	ob, err := c.GetOrderbook(ctx, "PF1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ob.Bids) != 1 || len(ob.Asks) != 1 {
		t.Fatalf("book bids=%d asks=%d", len(ob.Bids), len(ob.Asks))
	}

	if err := c.Authenticate(ctx); err != nil {
		t.Fatal(err)
	}

	orders, err := c.ListOrders(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders %+v", orders)
	}

	pos, err := c.GetPositions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pos) != 1 {
		t.Fatalf("pos %+v", pos)
	}

	bal, err := c.GetBalance(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bal["USDC"].Equal(decimal.NewFromInt(50)) {
		t.Fatalf("bal %+v", bal)
	}

	res, err := c.SendOrder(ctx, platforms.PlaceOrderRequest{MarketID: "PF1", Side: "yes", Size: decimal.NewFromInt(1), Price: decimal.RequireFromString("0.5")})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success || res.OrderID != "ord1" {
		t.Fatalf("order %+v", res)
	}

	if err := c.CancelOrder(ctx, "ord1"); err != nil {
		t.Fatal(err)
	}

	if err := c.HealthCheck(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestPredictFun_ErrNotConfigured(t *testing.T) {
	c := NewPredictFun("", "", "")
	ctx := context.Background()
	if _, err := c.GetAllMarkets(ctx); err != platforms.ErrNotConfigured {
		t.Fatalf("got %v", err)
	}
}
