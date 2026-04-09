package evmrpc

import (
	"context"
	"errors"
	"testing"
)

func TestPickFastestRPC_empty(t *testing.T) {
	_, _, err := PickFastestRPC(context.Background(), nil)
	if !errors.Is(err, ErrNoURLs) {
		t.Fatalf("got %v want ErrNoURLs", err)
	}
	_, _, err = PickFastestRPC(context.Background(), []string{"", "  "})
	if !errors.Is(err, ErrNoURLs) {
		t.Fatalf("got %v want ErrNoURLs", err)
	}
}

func TestNormalizeURLs_dedupe(t *testing.T) {
	got := normalizeURLs([]string{"https://a", "https://A", "https://b"})
	if len(got) != 2 || got[0] != "https://a" || got[1] != "https://b" {
		t.Fatalf("got %v", got)
	}
}
