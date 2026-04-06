package volatility

import (
	"math"
	"strings"
	"sync"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/ports/input"
	"github.com/shopspring/decimal"
)

// Tracker maintains per-asset EWMA of squared absolute returns (|Δmid|/mid) and maps it to a spread add-on.
// SpreadAddon updates state and returns the add-on in the same units as BaseSpread (probability 0–1 scale).
//
// Cold start: first observation per asset only records mid and returns 0 add-on (no spike).
// The add-on caps the contribution to the full bid–ask spread before halving in the quoting engine
// (i.e. it is added to base + vol_spread_bonus, then the sum is divided by two for half-spread).
type Tracker struct {
	mu      sync.Mutex
	cfg     config.MarketMakerCfg
	byAsset map[string]*assetState
}

type assetState struct {
	lastMid decimal.Decimal
	varEWMA float64 // EWMA of r^2, r = |mid-lastMid|/mid
}

var _ input.VolatilitySpread = (*Tracker)(nil)

// NewTracker builds a process-wide tracker (one instance per wiring root).
func NewTracker(cfg config.MarketMakerCfg) *Tracker {
	return &Tracker{cfg: cfg, byAsset: make(map[string]*assetState)}
}

// SpreadAddon implements input.VolatilitySpread.
func (t *Tracker) SpreadAddon(assetID string, mid decimal.Decimal) decimal.Decimal {
	if t == nil || t.cfg.EwmaVolSpreadScale <= 0 {
		return decimal.Zero
	}
	assetID = strings.TrimSpace(assetID)
	if assetID == "" || !mid.IsPositive() {
		return decimal.Zero
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.byAsset == nil {
		t.byAsset = make(map[string]*assetState)
	}
	st := t.byAsset[assetID]
	if st == nil {
		st = &assetState{}
		t.byAsset[assetID] = st
	}
	if !st.lastMid.IsPositive() {
		st.lastMid = mid
		return decimal.Zero
	}

	diff := mid.Sub(st.lastMid).Abs()
	r, _ := diff.Div(mid).Float64()
	st.lastMid = mid
	r2 := r * r

	lambda := t.cfg.EwmaVolLambda
	if lambda <= 0 || lambda >= 1 {
		lambda = 0.94
	}
	if st.varEWMA == 0 {
		st.varEWMA = r2
	} else {
		st.varEWMA = lambda*st.varEWMA + (1-lambda)*r2
	}

	addon := t.cfg.EwmaVolSpreadScale * math.Sqrt(st.varEWMA)
	if max := t.cfg.EwmaVolSpreadMax; max > 0 && addon > max {
		addon = max
	}
	return decimal.NewFromFloat(addon)
}
