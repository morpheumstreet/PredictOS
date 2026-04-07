package gabagool

import (
	"sync"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/api"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/executorclient"
	"github.com/shopspring/decimal"
)

type positionCache struct {
	fetchedAt time.Time
	byToken   map[string]decimal.Decimal
}

type PositionTracker struct {
	exec       *executorclient.Client
	mu         sync.Mutex
	cache      positionCache
	inventory  map[string]MarketInventory
}

func NewPositionTracker(ex *executorclient.Client) *PositionTracker {
	return &PositionTracker{
		exec:      ex,
		cache:     positionCache{byToken: map[string]decimal.Decimal{}},
		inventory: map[string]MarketInventory{},
	}
}

func (p *PositionTracker) RefreshIfStale() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Since(p.cache.fetchedAt) < 5*time.Second {
		return
	}
	pos, err := p.exec.GetPositions("", 500, 0)
	if err != nil {
		return
	}
	by := map[string]decimal.Decimal{}
	for _, x := range pos {
		by[x.Asset] = x.Size
	}
	p.cache = positionCache{fetchedAt: time.Now(), byToken: by}
}

func (p *PositionTracker) SyncInventory(markets []Market) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.cache.byToken) == 0 || len(markets) == 0 {
		return
	}
	for _, m := range markets {
		up := p.cache.byToken[m.UpTokenID]
		down := p.cache.byToken[m.DownTokenID]
		prev := p.inventory[m.Slug]
		p.inventory[m.Slug] = MarketInventory{
			UpShares: up, DownShares: down,
			LastUpFillAt: prev.LastUpFillAt, LastDownFillAt: prev.LastDownFillAt,
			LastUpFillPrice: prev.LastUpFillPrice, LastDownFillPrice: prev.LastDownFillPrice,
			LastTopUpAt: prev.LastTopUpAt,
		}
	}
}

func (p *PositionTracker) GetInventory(slug string) MarketInventory {
	p.mu.Lock()
	defer p.mu.Unlock()
	inv, ok := p.inventory[slug]
	if !ok {
		return EmptyInventory()
	}
	return inv
}

func (p *PositionTracker) AllInventories() map[string]MarketInventory {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]MarketInventory, len(p.inventory))
	for k, v := range p.inventory {
		out[k] = v
	}
	return out
}

func (p *PositionTracker) RecordFill(marketSlug string, isUp bool, shares, price decimal.Decimal) {
	p.mu.Lock()
	defer p.mu.Unlock()
	inv := p.inventory[marketSlug]
	at := time.Now()
	if isUp {
		pp := price
		inv = inv.AddUp(shares, at, &pp)
	} else {
		pp := price
		inv = inv.AddDown(shares, at, &pp)
	}
	p.inventory[marketSlug] = inv
}

func (p *PositionTracker) MarkTopUp(slug string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	inv := p.inventory[slug]
	p.inventory[slug] = inv.WithTopUp(time.Now())
}

// PositionsAsAPI returns cached positions for metrics (optional).
func (p *PositionTracker) PositionsAsAPI() []api.PolymarketPosition {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []api.PolymarketPosition
	for tid, sz := range p.cache.byToken {
		out = append(out, api.PolymarketPosition{Asset: tid, Size: sz})
	}
	return out
}
