package evmrpc

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// Manager owns a single *ethclient.Client, re-selected via PickFastestRPC when the connection
// is stale or after Invalidate (failover path). Call Close when done.
//
// Refresh is lazy (on Client): no background goroutine unless you add one at the app layer.
type Manager struct {
	urls          []string
	refreshEvery  time.Duration
	mu            sync.Mutex
	client        *ethclient.Client
	activeURL     string
	lastRefresh   time.Time
	forceNextDial bool
}

// NewManager copies urls (after normalization). refreshEvery <= 0 defaults to 5 minutes.
func NewManager(urls []string, refreshEvery time.Duration) *Manager {
	u := normalizeURLs(urls)
	if refreshEvery <= 0 {
		refreshEvery = 5 * time.Minute
	}
	cp := append([]string(nil), u...)
	return &Manager{urls: cp, refreshEvery: refreshEvery}
}

// Invalidate marks the current client for replacement on the next Client call (e.g. RPC error).
func (m *Manager) Invalidate() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.forceNextDial = true
	m.mu.Unlock()
}

// Close shuts down the underlying client.
func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}
	m.activeURL = ""
}

// Client returns the shared ethclient, dialing or re-probing when stale or invalidated.
func (m *Manager) Client(ctx context.Context) (*ethclient.Client, string, error) {
	if m == nil {
		return nil, "", ErrNoURLs
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.urls) == 0 {
		return nil, "", ErrNoURLs
	}
	if m.client != nil && !m.shouldRefreshLocked() {
		return m.client, m.activeURL, nil
	}
	if m.client != nil {
		m.client.Close()
		m.client = nil
		m.activeURL = ""
	}
	c, u, err := dialFastestFromNormalized(ctx, m.urls)
	if err != nil {
		return nil, "", err
	}
	m.client = c
	m.activeURL = u
	m.lastRefresh = time.Now()
	m.forceNextDial = false
	return c, u, nil
}

// ActiveURL returns the RPC URL for the current client (empty if none). For metrics / logs.
func (m *Manager) ActiveURL() string {
	if m == nil {
		return ""
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeURL
}

func (m *Manager) shouldRefreshLocked() bool {
	if m.forceNextDial {
		return true
	}
	if m.refreshEvery <= 0 {
		return false
	}
	return time.Since(m.lastRefresh) > m.refreshEvery
}
