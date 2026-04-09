package evmrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

func TestManager_Client_noURLs(t *testing.T) {
	m := NewManager(nil, time.Minute)
	_, _, err := m.Client(context.Background())
	if err != ErrNoURLs {
		t.Fatalf("got %v want ErrNoURLs", err)
	}
}

func TestManager_smokeWithURLs(t *testing.T) {
	m := NewManager([]string{"https://a.example", "https://b.example"}, time.Hour)
	m.Invalidate()
	m.Close()
}

func TestManager_Do_failoverSecondRPC(t *testing.T) {
	// Server A: fast web3_clientVersion, 503 on subsequent calls (e.g. eth_chainId).
	tsA := httptest.NewServer(jsonRPCHandler(t, rpcHandlerOpts{failNonVersion: true, slowClientVersion: false}))
	defer tsA.Close()
	// Server B: slow web3_clientVersion so A wins the initial probe; healthy for other methods.
	tsB := httptest.NewServer(jsonRPCHandler(t, rpcHandlerOpts{failNonVersion: false, slowClientVersion: true}))
	defer tsB.Close()

	ctx := context.Background()
	m := NewManager([]string{tsA.URL, tsB.URL}, time.Hour)
	t.Cleanup(func() { m.Close() })

	err := m.Do(ctx, 1, func(ctx context.Context, c *ethclient.Client) error {
		_, err := c.ChainID(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if active := m.ActiveURL(); active != tsB.URL {
		t.Fatalf("ActiveURL = %q want %q", active, tsB.URL)
	}
}

func TestManager_Do_noRetryReturnsFailoverErr(t *testing.T) {
	ts := httptest.NewServer(jsonRPCHandler(t, rpcHandlerOpts{failNonVersion: true, slowClientVersion: false}))
	defer ts.Close()

	ctx := context.Background()
	m := NewManager([]string{ts.URL}, time.Hour)
	t.Cleanup(func() { m.Close() })

	err := m.Do(ctx, 0, func(ctx context.Context, c *ethclient.Client) error {
		_, err := c.ChainID(ctx)
		return err
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsFailoverError(err) {
		t.Fatalf("expected failover error, got %v", err)
	}
}

type rpcHandlerOpts struct {
	failNonVersion    bool
	slowClientVersion bool
}

func jsonRPCHandler(t *testing.T, opts rpcHandlerOpts) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(req.ID) == 0 {
			req.ID = []byte(`0`)
		}
		switch req.Method {
		case "web3_clientVersion":
			if opts.slowClientVersion {
				time.Sleep(200 * time.Millisecond)
			}
			writeJSONRPCResult(w, req.ID, `"test/1.0"`)
		case "eth_chainId":
			if opts.failNonVersion {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			writeJSONRPCResult(w, req.ID, `"0x89"`)
		default:
			if opts.failNonVersion {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			writeJSONRPCResult(w, req.ID, `"0x0"`)
		}
	}
}

func writeJSONRPCResult(w http.ResponseWriter, id json.RawMessage, resultJSON string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, string(id), resultJSON)
}
