package evmrpc

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchHTTPSRPCsForChain(t *testing.T) {
	const body = `[
  {"chainId": 1, "rpc": [{"url": "https://eth.example/rpc"}]},
  {"chainId": 137, "rpc": [
    {"url": "wss://skip.me"},
    {"url": "https://polygon.a/rpc"},
    {"url": "https://polygon.b/rpc"},
    {"url": "https://polygon.a/rpc"}
  ]}
]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	got, err := FetchHTTPSRPCsForChain(context.Background(), srv.Client(), srv.URL, 137, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "https://polygon.a/rpc" || got[1] != "https://polygon.b/rpc" {
		t.Fatalf("got %v", got)
	}
}

func TestFetchHTTPSRPCsForChain_maxN(t *testing.T) {
	var b bytes.Buffer
	b.WriteString(`[{"chainId":137,"rpc":[`)
	for i := 0; i < 20; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		_, _ = fmt.Fprintf(&b, `{"url":"https://h%d.example"}`, i)
	}
	b.WriteString(`]}]`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(b.Bytes())
	}))
	defer srv.Close()

	got, err := FetchHTTPSRPCsForChain(context.Background(), srv.Client(), srv.URL, 137, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("len %d", len(got))
	}
}
