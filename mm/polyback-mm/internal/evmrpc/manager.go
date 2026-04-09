package evmrpc

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// DefaultFailoverBanDuration is how long an RPC URL is skipped from selection after a
// failover-triggering error on that endpoint.
const DefaultFailoverBanDuration = 30 * time.Second

// Manager owns a single *ethclient.Client, re-selected via PickFastestRPC when the connection
// is stale, after Invalidate (failover path), or when [Manager.Do] bans a bad URL. Call Close when done.
//
// Refresh is lazy (on Client): no background goroutine unless you add one at the app layer.
type Manager struct {
	urls             []string
	refreshEvery     time.Duration
	failoverCooldown time.Duration
	mu               sync.Mutex
	client           *ethclient.Client
	activeURL        string
	lastRefresh      time.Time
	forceNextDial    bool
	banUntil         map[string]time.Time
}

// NewManager copies urls (after normalization). refreshEvery <= 0 defaults to 5 minutes.
func NewManager(urls []string, refreshEvery time.Duration) *Manager {
	u := normalizeURLs(urls)
	if refreshEvery <= 0 {
		refreshEvery = 5 * time.Minute
	}
	cp := append([]string(nil), u...)
	return &Manager{
		urls:             cp,
		refreshEvery:     refreshEvery,
		failoverCooldown: DefaultFailoverBanDuration,
	}
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
// On dial failure after a winning probe, the second return value is that URL (for banning); otherwise it is empty.
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
	eligible := m.eligibleURLsLocked()
	c, u, err := dialFastestFromNormalized(ctx, eligible)
	if err != nil {
		return nil, u, err
	}
	m.client = c
	m.activeURL = u
	m.lastRefresh = time.Now()
	m.forceNextDial = false
	return c, u, nil
}

// Do runs fn with a client from [Manager.Client]. On [IsFailoverError], it bans the active URL
// (see [DefaultFailoverBanDuration]), calls [Manager.Invalidate], and retries if attempt < maxRetries.
//
// maxRetries is the number of failover retries after the first attempt (maxRetries==0 → one try only;
// maxRetries==1 → initial try plus one failover).
func (m *Manager) Do(ctx context.Context, maxRetries int, fn func(context.Context, *ethclient.Client) error) error {
	if m == nil {
		return ErrNoURLs
	}
	if maxRetries < 0 {
		maxRetries = 0
	}
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		c, url, err := m.Client(ctx)
		if err != nil {
			lastErr = err
			if IsFailoverError(err) && attempt < maxRetries {
				m.banAndInvalidate(url)
				continue
			}
			return err
		}
		err = fn(ctx, c)
		if err == nil {
			return nil
		}
		lastErr = err
		if !IsFailoverError(err) {
			return err
		}
		if attempt == maxRetries {
			return err
		}
		m.banAndInvalidate(url)
	}
	return lastErr
}

func (m *Manager) banAndInvalidate(url string) {
	m.mu.Lock()
	if url != "" {
		m.banURLLocked(url)
	}
	m.forceNextDial = true
	m.mu.Unlock()
}

func (m *Manager) banURLLocked(u string) {
	if u == "" {
		return
	}
	if m.banUntil == nil {
		m.banUntil = make(map[string]time.Time)
	}
	d := m.failoverCooldown
	if d <= 0 {
		d = DefaultFailoverBanDuration
	}
	m.banUntil[u] = time.Now().Add(d)
}

// eligibleURLsLocked returns URLs that are not within their post-failover ban window.
// If every URL is banned, returns a copy of the full list so dialing can still proceed.
func (m *Manager) eligibleURLsLocked() []string {
	if len(m.urls) == 0 {
		return nil
	}
	now := time.Now()
	var out []string
	for _, u := range m.urls {
		if until, ok := m.banUntil[u]; ok && now.Before(until) {
			continue
		}
		out = append(out, u)
	}
	if len(out) == 0 {
		return append([]string(nil), m.urls...)
	}
	return out
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
