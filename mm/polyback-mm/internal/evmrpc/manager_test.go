package evmrpc

import (
	"context"
	"testing"
	"time"
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
