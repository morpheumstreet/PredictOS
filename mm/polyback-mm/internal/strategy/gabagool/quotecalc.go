package gabagool

import (
	"math"
	"strings"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/ws"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/metrics"
	"github.com/shopspring/decimal"
)

type QuoteCalculator struct {
	bank *Bankroll
	cfg  *config.Root
	met  *metrics.Service
}

func NewQuoteCalculator(b *Bankroll, root *config.Root, m *metrics.Service) *QuoteCalculator {
	return &QuoteCalculator{bank: b, cfg: root, met: m}
}

func (q *QuoteCalculator) CalculateEntryPrice(book *polyws.TopOfBook, tickSize decimal.Decimal, g *config.GabagoolCfg, skewTicks int) *decimal.Decimal {
	if book == nil || book.BestBid == nil || book.BestAsk == nil {
		return nil
	}
	bb, ba := *book.BestBid, *book.BestAsk
	mid := bb.Add(ba).Div(decimal.NewFromInt(2)).Round(4)
	spread := ba.Sub(bb)
	improve := g.ImproveTicks
	effectiveImprove := improve + skewTicks

	var entry decimal.Decimal
	six := decimal.NewFromFloat(0.06)
	if spread.GreaterThanOrEqual(six) {
		adj := math.Max(0, float64(improve-skewTicks))
		entry = mid.Sub(tickSize.Mul(decimal.NewFromFloat(adj)))
	} else {
		improvedBid := bb.Add(tickSize.Mul(decimal.NewFromInt(int64(effectiveImprove))))
		if improvedBid.LessThan(mid) {
			entry = improvedBid
		} else {
			entry = mid
		}
	}
	entry = RoundToTick(entry, tickSize, false)
	if entry.LessThan(decimal.NewFromFloat(0.01)) {
		return nil
	}
	if entry.GreaterThan(decimal.NewFromFloat(0.99)) {
		return nil
	}
	if entry.GreaterThanOrEqual(ba) {
		entry = ba.Sub(tickSize)
		if entry.LessThan(decimal.NewFromFloat(0.01)) {
			return nil
		}
	}
	return &entry
}

func RoundToTick(value, tickSize decimal.Decimal, up bool) decimal.Decimal {
	if !tickSize.IsPositive() {
		return value
	}
	ticks := value.Div(tickSize)
	var r decimal.Decimal
	if up {
		r = ticks.Ceil()
	} else {
		r = ticks.Floor()
	}
	return r.Mul(tickSize)
}

func (q *QuoteCalculator) CalculateShares(m *Market, entry decimal.Decimal, g *config.GabagoolCfg, secondsToEnd int64, exposure decimal.Decimal) *decimal.Decimal {
	rep := replicaSharesByTimeToEnd(m, secondsToEnd)
	var shares decimal.Decimal
	if rep != nil {
		shares = *rep
	} else {
		n := q.calculateNotional(g, exposure)
		if n == nil {
			return nil
		}
		sh := n.Div(entry).RoundDown(2)
		if sh.LessThan(decimal.NewFromFloat(0.01)) {
			return nil
		}
		return &sh
	}
	if !entry.IsPositive() {
		return nil
	}
	shares = shares.Mul(q.bank.DynamicSizingMultiplier(g))
	bankUsd := q.bank.ResolveEffective(g)
	if bankUsd.IsPositive() {
		if g.MaxOrderBankrollFraction > 0 {
			capUsd := bankUsd.Mul(decimal.NewFromFloat(g.MaxOrderBankrollFraction))
			capSh := capUsd.Div(entry).RoundDown(2)
			if capSh.LessThan(shares) {
				shares = capSh
			}
		}
		if g.MaxTotalBankrollFraction > 0 {
			totalCap := bankUsd.Mul(decimal.NewFromFloat(g.MaxTotalBankrollFraction))
			rem := totalCap.Sub(exposure)
			if !rem.IsPositive() {
				return nil
			}
			capSh := rem.Div(entry).RoundDown(2)
			if capSh.LessThan(shares) {
				shares = capSh
			}
		}
	}
	if mx := q.cfg.Hft.Risk.MaxOrderNotionalUsd; mx > 0 {
		capSh := decimal.NewFromFloat(mx).Div(entry).RoundDown(2)
		if capSh.LessThan(shares) {
			shares = capSh
		}
	}
	shares = shares.RoundDown(2)
	if shares.LessThan(decimal.NewFromFloat(0.01)) {
		return nil
	}
	return &shares
}

func (q *QuoteCalculator) calculateNotional(g *config.GabagoolCfg, exposure decimal.Decimal) *decimal.Decimal {
	bankUsd := q.bank.ResolveEffective(g)
	var notional decimal.Decimal
	if bankUsd.IsPositive() && g.QuoteSizeBankrollFraction > 0 {
		notional = bankUsd.Mul(decimal.NewFromFloat(g.QuoteSizeBankrollFraction))
	} else {
		notional = decimal.NewFromFloat(g.QuoteSize)
	}
	if !notional.IsPositive() {
		return nil
	}
	if mx := q.cfg.Hft.Risk.MaxOrderNotionalUsd; mx > 0 {
		mxd := decimal.NewFromFloat(mx)
		if mxd.LessThan(notional) {
			notional = mxd
		}
	}
	if bankUsd.IsPositive() {
		if g.MaxOrderBankrollFraction > 0 {
			cap := bankUsd.Mul(decimal.NewFromFloat(g.MaxOrderBankrollFraction))
			if cap.LessThan(notional) {
				notional = cap
			}
		}
		if g.MaxTotalBankrollFraction > 0 {
			rem := bankUsd.Mul(decimal.NewFromFloat(g.MaxTotalBankrollFraction)).Sub(exposure)
			if !rem.IsPositive() {
				return nil
			}
			if rem.LessThan(notional) {
				notional = rem
			}
		}
	}
	if !notional.IsPositive() {
		return nil
	}
	return &notional
}

func replicaSharesByTimeToEnd(m *Market, secondsToEnd int64) *decimal.Decimal {
	if m == nil {
		return nil
	}
	slug := m.Slug
	switch {
	case strings.HasPrefix(slug, "btc-updown-15m-"):
		return pickReplica(secondsToEnd, 11, 13, 17, 19, 20)
	case strings.HasPrefix(slug, "eth-updown-15m-"):
		return pickReplica(secondsToEnd, 8, 10, 12, 13, 14)
	case strings.HasPrefix(slug, "bitcoin-up-or-down-"):
		return pickReplica6(secondsToEnd, 9, 10, 11, 12, 14, 15, 17, 18)
	case strings.HasPrefix(slug, "ethereum-up-or-down-"):
		return pickReplica6(secondsToEnd, 7, 8, 9, 11, 12, 13, 14, 14)
	}
	return nil
}

func pickReplica(sec int64, a, b, c, d, e int) *decimal.Decimal {
	switch {
	case sec < 60:
		return dec(a)
	case sec < 180:
		return dec(b)
	case sec < 300:
		return dec(c)
	case sec < 600:
		return dec(d)
	default:
		return dec(e)
	}
}

func pickReplica6(sec int64, v ...int) *decimal.Decimal {
	// v has 8 thresholds matching Java
	if len(v) < 8 {
		return nil
	}
	switch {
	case sec < 60:
		return dec(v[0])
	case sec < 180:
		return dec(v[1])
	case sec < 300:
		return dec(v[2])
	case sec < 600:
		return dec(v[3])
	case sec < 900:
		return dec(v[4])
	case sec < 1200:
		return dec(v[5])
	case sec < 1800:
		return dec(v[6])
	default:
		return dec(v[7])
	}
}

func dec(i int) *decimal.Decimal {
	d := decimal.NewFromInt(int64(i))
	return &d
}

func (q *QuoteCalculator) CalculateSkewTicks(inv MarketInventory, g *config.GabagoolCfg) (up, down int) {
	imb := inv.Imbalance()
	maxSkew := g.CompleteSetMaxSkewTicks
	th := decimal.NewFromFloat(g.CompleteSetImbalanceSharesForMaxSkew)
	if !th.IsPositive() || maxSkew <= 0 {
		return 0, 0
	}
	ratio := math.Min(1, math.Abs(mustFloat(imb))/mustFloat(th))
	skew := int(math.Round(ratio * float64(maxSkew)))
	if imb.IsPositive() {
		return -skew, skew
	}
	if imb.IsNegative() {
		return skew, -skew
	}
	return 0, 0
}

func mustFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

func (q *QuoteCalculator) CalculateExposure(open map[string]*OrderState, inventories map[string]MarketInventory) decimal.Decimal {
	openN := decimal.Zero
	for _, o := range open {
		if o == nil {
			continue
		}
		rem := o.Size.Sub(o.MatchedSize)
		if rem.IsNegative() {
			rem = decimal.Zero
		}
		openN = openN.Add(o.Price.Mul(rem))
	}
	unhedged := decimal.Zero
	for _, inv := range inventories {
		imb := inv.Imbalance().Abs()
		if imb.IsPositive() {
			unhedged = unhedged.Add(imb.Mul(decimal.NewFromFloat(0.5)))
		}
	}
	total := openN.Add(unhedged)
	if q.met != nil {
		q.met.UpdateTotalExposure(total)
	}
	return total
}

// ValidateMakerBidPrice applies gabagool bounds and non-crossing vs best ask (same as CalculateEntryPrice tail).
func ValidateMakerBidPrice(p decimal.Decimal, book *polyws.TopOfBook, tick decimal.Decimal) *decimal.Decimal {
	if book == nil {
		return nil
	}
	if p.LessThan(decimal.NewFromFloat(0.01)) || p.GreaterThan(decimal.NewFromFloat(0.99)) {
		return nil
	}
	if book.BestAsk != nil && p.GreaterThanOrEqual(*book.BestAsk) {
		adj := book.BestAsk.Sub(tick)
		if adj.LessThan(decimal.NewFromFloat(0.01)) {
			return nil
		}
		return &adj
	}
	return &p
}

func (q *QuoteCalculator) HasMinimumEdge(upPrice, downPrice decimal.Decimal, g *config.GabagoolCfg) bool {
	cost := upPrice.Add(downPrice)
	edge := decimal.NewFromInt(1).Sub(cost)
	return edge.GreaterThanOrEqual(decimal.NewFromFloat(g.CompleteSetMinEdge))
}
