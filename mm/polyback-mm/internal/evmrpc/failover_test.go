package evmrpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
)

type stubRPCError struct {
	code int
	msg  string
}

func (e stubRPCError) Error() string   { return e.msg }
func (e stubRPCError) ErrorCode() int { return e.code }

func TestIsFailoverError(t *testing.T) {
	t.Parallel()
	opErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"canceled", context.Canceled, false},
		{"wrappedCanceled", fmt.Errorf("x: %w", context.Canceled), false},
		{"deadline", context.DeadlineExceeded, true},
		{"wrappedDeadline", fmt.Errorf("x: %w", context.DeadlineExceeded), true},
		{"eof", io.EOF, true},
		{"wrappedEOF", fmt.Errorf("rpc: %w", io.EOF), true},
		{"clientQuit", rpc.ErrClientQuit, true},
		{"http503", rpc.HTTPError{StatusCode: http.StatusServiceUnavailable}, true},
		{"http429", rpc.HTTPError{StatusCode: http.StatusTooManyRequests}, true},
		{"http408", rpc.HTTPError{StatusCode: http.StatusRequestTimeout}, true},
		{"http400", rpc.HTTPError{StatusCode: http.StatusBadRequest}, false},
		{"opError", opErr, true},
		{"rpcTimeoutCode", stubRPCError{code: -32002, msg: "timeout"}, true},
		{"rpcTooLarge", stubRPCError{code: -32003, msg: "large"}, true},
		{"rpcInvalidParams", stubRPCError{code: -32602, msg: "params"}, false},
		{"rpcRevertish", stubRPCError{code: -32603, msg: "execution reverted"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsFailoverError(tc.err); got != tc.want {
				t.Fatalf("IsFailoverError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
