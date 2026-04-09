package evmrpc

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"
)

// IsFailoverError reports whether err is an infrastructure / transport class error for which
// rotating to another JSON-RPC endpoint may help. It returns false for nil, context.Canceled,
// JSON-RPC client/parameter errors, and typical execution-revert style failures.
//
// Prefer [Manager.Do] for automatic ban + invalidate + retry; use IsFailoverError when handling
// errors from a raw [*ethclient.Client] obtained via [Manager.Client].
func IsFailoverError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	if errors.Is(err, rpc.ErrClientQuit) {
		return true
	}
	var he rpc.HTTPError
	if errors.As(err, &he) {
		return httpFailoverStatus(he.StatusCode)
	}
	var op *net.OpError
	if errors.As(err, &op) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	var re rpc.Error
	if errors.As(err, &re) {
		switch re.ErrorCode() {
		case -32002, -32003: // request timed out, response too large (geth rpc)
			return true
		default:
			return false
		}
	}
	return false
}

func httpFailoverStatus(code int) bool {
	if code >= 500 {
		return true
	}
	switch code {
	case http.StatusRequestTimeout, http.StatusTooManyRequests: // 408, 429
		return true
	default:
		return false
	}
}
