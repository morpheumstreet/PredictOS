package tracker

import (
	"fmt"
	"os"
	"strings"
)

// MaxTrackerWallets caps batch size per request (public data-api friendly).
const MaxTrackerWallets = 32

var walletRequestKeysSingle = []string{"address", "user", "wallet"}

// parseWalletAddresses extracts 0x…40-hex wallets from the JSON body (like polymarket-trade-tracker per-query address).
// Accepts: single string fields address | user | wallet; arrays addresses | wallets.
// If the body has no wallets, falls back to POLYMARKET_PROXY_WALLET_ADDRESS for backward compatibility.
func parseWalletAddresses(req map[string]any) ([]string, error) {
	seen := make(map[string]struct{})
	var out []string

	add := func(raw string) error {
		a := strings.TrimSpace(raw)
		if a == "" {
			return nil
		}
		if !isValidEVMAddress(a) {
			return fmt.Errorf("invalid wallet address: %q", raw)
		}
		key := strings.ToLower(a)
		if _, ok := seen[key]; ok {
			return nil
		}
		seen[key] = struct{}{}
		out = append(out, a)
		if len(out) > MaxTrackerWallets {
			return fmt.Errorf("too many wallets (max %d)", MaxTrackerWallets)
		}
		return nil
	}

	for _, k := range walletRequestKeysSingle {
		if s, ok := req[k].(string); ok {
			if err := add(s); err != nil {
				return nil, err
			}
		}
	}

	for _, arrKey := range []string{"addresses", "wallets"} {
		if arr, ok := req[arrKey].([]any); ok {
			for _, v := range arr {
				s, ok := v.(string)
				if !ok {
					continue
				}
				if err := add(s); err != nil {
					return nil, err
				}
			}
		}
	}

	if len(out) == 0 {
		if env := strings.TrimSpace(os.Getenv("POLYMARKET_PROXY_WALLET_ADDRESS")); env != "" {
			if err := add(env); err != nil {
				return nil, err
			}
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no wallet address: set address, addresses, or POLYMARKET_PROXY_WALLET_ADDRESS")
	}
	return out, nil
}

func isValidEVMAddress(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 42 || !strings.HasPrefix(s, "0x") {
		return false
	}
	for i := 2; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' {
			continue
		}
		return false
	}
	return true
}
