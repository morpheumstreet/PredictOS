// Package evmrpc provides JSON-RPC endpoint probing and client dial with latency-based selection,
// following the approach in morpheum-labs/pricefeeding rpcscan (parallel probes, pick fastest).
package evmrpc

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// DefaultPerURLProbe is the timeout for each endpoint when probing (avoid hanging on dead RPCs).
const DefaultPerURLProbe = 5 * time.Second

// ErrNoURLs is returned when no RPC endpoints are configured.
var ErrNoURLs = errors.New("evmrpc: no rpc urls")

// PickFastestRPC probes all HTTPS/WSS URLs in parallel and returns the lowest-latency endpoint
// that successfully answers web3_clientVersion. Empty or duplicate URLs are skipped.
func PickFastestRPC(ctx context.Context, urls []string) (url string, latency time.Duration, err error) {
	return pickFastestFromNormalized(ctx, normalizeURLs(urls))
}

// pickFastestFromNormalized runs parallel probes; urls must already be normalizeURLs output.
func pickFastestFromNormalized(ctx context.Context, urls []string) (url string, latency time.Duration, err error) {
	if len(urls) == 0 {
		return "", 0, ErrNoURLs
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var mu sync.Mutex
	bestURL := ""
	bestLat := time.Duration(math.MaxInt64)
	var wg sync.WaitGroup
	for _, u := range urls {
		u := u
		wg.Add(1)
		go func() {
			defer wg.Done()
			lat, perr := probe(ctx, u, DefaultPerURLProbe)
			if perr != nil {
				return
			}
			mu.Lock()
			if lat < bestLat {
				bestLat, bestURL = lat, u
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if bestURL == "" {
		return "", 0, fmt.Errorf("evmrpc: no reachable rpc among %d candidates", len(urls))
	}
	return bestURL, bestLat, nil
}

// DialFastest picks the fastest URL then dials an ethclient. The returned client must be closed by the caller.
func DialFastest(ctx context.Context, urls []string) (*ethclient.Client, string, error) {
	return dialFastestFromNormalized(ctx, normalizeURLs(urls))
}

// dialFastestFromNormalized assumes urls are already normalizeURLs output (Manager holds normalized slice).
func dialFastestFromNormalized(ctx context.Context, urls []string) (*ethclient.Client, string, error) {
	u, _, err := pickFastestFromNormalized(ctx, urls)
	if err != nil {
		return nil, "", err
	}
	c, err := ethclient.DialContext(ctx, u)
	if err != nil {
		return nil, "", err
	}
	return c, u, nil
}

func probe(parent context.Context, rawURL string, timeout time.Duration) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	start := time.Now()
	c, err := rpc.DialContext(ctx, rawURL)
	if err != nil {
		return 0, err
	}
	defer c.Close()
	var v string
	if err := c.CallContext(ctx, &v, "web3_clientVersion"); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

func normalizeURLs(urls []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		key := strings.ToLower(u)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, u)
	}
	return out
}
