package gabagool

import (
	"strings"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/executorclient"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/metrics"
	"github.com/shopspring/decimal"
)

type bankrollSnapshot struct {
	fetchedAt       time.Time
	usdc            decimal.Decimal
	equity          decimal.Decimal
	smoothedUsdc    decimal.Decimal
	smoothedEquity  decimal.Decimal
}

type Bankroll struct {
	cfg    *config.Root
	exec   *executorclient.Client
	met    *metrics.Service
	cache  bankrollSnapshot
}

func NewBankroll(root *config.Root, ex *executorclient.Client, m *metrics.Service) *Bankroll {
	return &Bankroll{cfg: root, exec: ex, met: m}
}

func (b *Bankroll) RefreshIfStale(g *config.GabagoolCfg) {
	if g == nil {
		return
	}
	now := time.Now()
	if strings.EqualFold(g.BankrollMode, "FIXED") {
		if b.met != nil {
			b.met.UpdateBankroll(decimal.NewFromFloat(g.BankrollUsd))
		}
		return
	}
	refreshMs := g.BankrollRefreshMillis
	if refreshMs < 1000 {
		refreshMs = 1000
	}
	if now.Sub(b.cache.fetchedAt) < time.Duration(refreshMs)*time.Millisecond {
		if eff := b.ResolveEffective(g); b.met != nil {
			b.met.UpdateBankroll(eff)
		}
		return
	}
	resp, err := b.exec.GetBankroll()
	if err != nil {
		return
	}
	usdc := resp.USDCBalance
	equity := resp.TotalEquityUsd
	if equity.IsZero() {
		equity = usdc
	}
	alpha := g.BankrollSmoothingAlpha
	if alpha < 0.01 {
		alpha = 0.01
	}
	if alpha > 1 {
		alpha = 1
	}
	a := decimal.NewFromFloat(alpha)
	one := decimal.NewFromInt(1)
	oma := one.Sub(a)
	prevU := b.cache.smoothedUsdc
	if prevU.IsZero() {
		prevU = usdc
	}
	prevE := b.cache.smoothedEquity
	if prevE.IsZero() {
		prevE = equity
	}
	su := usdc.Mul(a).Add(prevU.Mul(oma))
	se := equity.Mul(a).Add(prevE.Mul(oma))
	b.cache = bankrollSnapshot{
		fetchedAt: now, usdc: usdc, equity: equity, smoothedUsdc: su, smoothedEquity: se,
	}
	if eff := b.ResolveEffective(g); b.met != nil {
		b.met.UpdateBankroll(eff)
	}
}

func (b *Bankroll) ResolveEffective(g *config.GabagoolCfg) decimal.Decimal {
	if g == nil {
		return decimal.Zero
	}
	if strings.EqualFold(g.BankrollMode, "FIXED") {
		return decimal.NewFromFloat(g.BankrollUsd)
	}
	eq := b.cache.smoothedEquity
	if eq.IsZero() {
		eq = b.cache.equity
	}
	if eq.IsZero() {
		return decimal.Zero
	}
	frac := g.BankrollTradingFraction
	if frac <= 0 {
		frac = 1
	}
	return eq.Mul(decimal.NewFromFloat(frac))
}

func (b *Bankroll) IsBelowThreshold(g *config.GabagoolCfg) bool {
	if g == nil {
		return false
	}
	th := decimal.NewFromFloat(g.BankrollMinThreshold)
	if th.IsZero() {
		return false
	}
	return b.ResolveEffective(g).LessThan(th)
}

func (b *Bankroll) DynamicSizingMultiplier(g *config.GabagoolCfg) decimal.Decimal {
	eff := b.ResolveEffective(g)
	if eff.IsZero() {
		return decimal.NewFromInt(1)
	}
	base := decimal.NewFromFloat(g.BankrollUsd)
	if base.IsZero() {
		return decimal.NewFromInt(1)
	}
	r := eff.Div(base)
	if r.LessThan(decimal.NewFromFloat(0.25)) {
		return decimal.NewFromFloat(0.25)
	}
	if r.GreaterThan(decimal.NewFromInt(4)) {
		return decimal.NewFromInt(4)
	}
	return r
}
