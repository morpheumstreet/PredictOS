package ws

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"
)

// EventListener optional HTTP poll: first successful response sets baseline hash; later body changes invoke OnAlert.
type EventListener struct {
	PollURL      string
	PollInterval time.Duration
	HTTPClient   *http.Client
	OnAlert      func()
	lastHash     string
}

// Start blocks until ctx is cancelled. No-op when PollURL is empty.
func (e *EventListener) Start(ctx context.Context) {
	if e == nil || strings.TrimSpace(e.PollURL) == "" {
		<-ctx.Done()
		return
	}
	client := e.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	interval := e.PollInterval
	if interval <= 0 {
		interval = 60 * time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	e.pollOnce(client)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			e.pollOnce(client)
		}
	}
}

func (e *EventListener) pollOnce(client *http.Client) {
	if e == nil {
		return
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimSpace(e.PollURL), nil)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return
	}
	sum := sha256.Sum256(body)
	h := hex.EncodeToString(sum[:])
	if e.lastHash == "" {
		e.lastHash = h
		return
	}
	if h != e.lastHash && e.OnAlert != nil {
		e.OnAlert()
	}
	e.lastHash = h
}
