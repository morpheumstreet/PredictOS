package tracker

import (
	"testing"
)

func TestParseWalletAddresses_bodySingle(t *testing.T) {
	req := map[string]any{"address": "0xabcdef0123456789abcdef0123456789abcdef01"}
	got, err := parseWalletAddresses(req)
	if err != nil || len(got) != 1 || got[0] != req["address"] {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestParseWalletAddresses_aliasesAndDedupe(t *testing.T) {
	a := "0xabcdef0123456789abcdef0123456789abcdef01"
	req := map[string]any{
		"addresses": []any{a, "0xABCDEF0123456789ABCDEF0123456789ABCDEF01", "0x1111111111111111111111111111111111111111"},
	}
	got, err := parseWalletAddresses(req)
	if err != nil || len(got) != 2 {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestParseWalletAddresses_envFallback(t *testing.T) {
	t.Setenv("POLYMARKET_PROXY_WALLET_ADDRESS", "0x2222222222222222222222222222222222222222")
	got, err := parseWalletAddresses(map[string]any{})
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v err %v", got, err)
	}
}

func TestParseWalletAddresses_invalid(t *testing.T) {
	_, err := parseWalletAddresses(map[string]any{"address": "not-an-address"})
	if err == nil {
		t.Fatal("expected error")
	}
}
