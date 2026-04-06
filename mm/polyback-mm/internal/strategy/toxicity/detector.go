package toxicity

import (
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/ports/input"
	"github.com/shopspring/decimal"
)

// Detector implements ports/input.ToxicityDetector.
type Detector struct {
	cfg config.MarketMakerCfg
}

var _ input.ToxicityDetector = (*Detector)(nil)

func NewDetector(cfg config.MarketMakerCfg) *Detector {
	return &Detector{cfg: cfg}
}

func (d *Detector) Assess(trades []domain.Trade, book *domain.OrderBookL2) domain.ToxicitySignal {
	if book == nil {
		return domain.ToxicitySignal{}
	}
	win := d.window()
	cutoff := time.Now().UTC().Add(-win)
	var inWin []domain.Trade
	for i := len(trades) - 1; i >= 0; i-- {
		t := trades[i]
		if t.Timestamp.Before(cutoff) {
			break
		}
		inWin = append(inWin, t)
	}
	// restore chronological order
	for i, j := 0, len(inWin)-1; i < j; i, j = i+1, j-1 {
		inWin[i], inWin[j] = inWin[j], inWin[i]
	}

	burstN := d.burstThreshold()
	burst := len(inWin)
	level := domain.ToxicityNone
	if burst >= burstN {
		level = domain.ToxicityElevated
	}
	unsafeBurst := d.unsafeBurst()
	if burst >= unsafeBurst {
		return domain.ToxicitySignal{
			Level: domain.ToxicityHigh, BurstTrades: burst, Unsafe: true,
			BidPenalty: d.maxPenalty(), AskPenalty: d.maxPenalty(),
		}
	}

	mid := midFromBook(book)
	sp := spreadFromBook(book)
	impact := d.impactScore(mid, sp, inWin)
	if impact >= d.impactThreshold(sp) && burst >= 2 {
		if level < domain.ToxicityElevated {
			level = domain.ToxicityElevated
		}
	}

	dropBid, dropAsk := liquidityDrops(book, d.dropRatio())
	if dropBid > 0 || dropAsk > 0 {
		if level < domain.ToxicityElevated {
			level = domain.ToxicityElevated
		}
	}

	pen := decimal.Zero
	if level >= domain.ToxicityElevated {
		pen = d.penaltyFor(level, burst, impact, dropBid, dropAsk)
	}
	if level == domain.ToxicityHigh {
		pen = d.maxPenalty()
	}

	return domain.ToxicitySignal{
		Level:            level,
		BurstTrades:      burst,
		ImpactScore:      impact,
		LiquidityDropBid: dropBid,
		LiquidityDropAsk: dropAsk,
		BidPenalty:       pen,
		AskPenalty:       pen,
	}
}

func (d *Detector) window() time.Duration {
	ms := d.cfg.TradeWindowMillis
	if ms <= 0 {
		ms = 2000
	}
	return time.Duration(ms) * time.Millisecond
}

func (d *Detector) burstThreshold() int {
	n := d.cfg.BurstTradeCount
	if n <= 0 {
		return 5
	}
	return n
}

func (d *Detector) unsafeBurst() int {
	n := d.cfg.ToxicityUnsafeBurst
	if n <= 0 {
		return 12
	}
	return n
}

func (d *Detector) dropRatio() float64 {
	r := d.cfg.LiquidityDropRatio
	if r <= 0 {
		return 0.35
	}
	return r
}

func (d *Detector) maxPenalty() decimal.Decimal {
	m := d.cfg.ToxicityPenaltyMax
	if m <= 0 {
		m = 0.015
	}
	return decimal.NewFromFloat(m)
}

func (d *Detector) impactThreshold(spread decimal.Decimal) float64 {
	mult := d.cfg.ImpactSpreadMultiple
	if mult <= 0 {
		mult = 2
	}
	f, _ := spread.Mul(decimal.NewFromFloat(mult)).Float64()
	return f
}

func (d *Detector) penaltyFor(level domain.ToxicityLevel, burst int, impact, dropBid, dropAsk float64) decimal.Decimal {
	maxP := d.maxPenalty()
	if level == domain.ToxicityHigh {
		return maxP
	}
	frac := 0.25
	if burst >= d.burstThreshold()*2 {
		frac += 0.15
	}
	if impact > 0 {
		frac += 0.15
	}
	if dropBid > 0 || dropAsk > 0 {
		frac += 0.1
	}
	if frac > 1 {
		frac = 1
	}
	return maxP.Mul(decimal.NewFromFloat(frac))
}

func midFromBook(b *domain.OrderBookL2) decimal.Decimal {
	if b == nil || b.BestBid == nil || b.BestAsk == nil {
		return decimal.Zero
	}
	return b.BestBid.Add(*b.BestAsk).Div(decimal.NewFromInt(2))
}

func spreadFromBook(b *domain.OrderBookL2) decimal.Decimal {
	if b == nil || b.BestBid == nil || b.BestAsk == nil {
		return decimal.NewFromFloat(0.01)
	}
	s := b.BestAsk.Sub(*b.BestBid)
	if !s.IsPositive() {
		return decimal.NewFromFloat(0.01)
	}
	return s
}

func (d *Detector) impactScore(mid, spread decimal.Decimal, inWin []domain.Trade) float64 {
	if len(inWin) == 0 || !mid.IsPositive() {
		return 0
	}
	var sum decimal.Decimal
	var n int
	for _, t := range inWin {
		sum = sum.Add(t.Price)
		n++
	}
	if n == 0 {
		return 0
	}
	avg := sum.Div(decimal.NewFromInt(int64(n)))
	diff := mid.Sub(avg).Abs()
	f, _ := diff.Float64()
	return f
}

func liquidityDrops(b *domain.OrderBookL2, ratio float64) (bidDrop, askDrop float64) {
	if b == nil || ratio <= 0 || ratio >= 1 {
		return 0, 0
	}
	if b.BestBidSize != nil && b.EMABidSize != nil && b.EMABidSize.IsPositive() {
		cur, _ := b.BestBidSize.Float64()
		ema, _ := b.EMABidSize.Float64()
		if ema > 0 && cur < ema*(1-ratio) {
			bidDrop = 1 - cur/ema
		}
	}
	if b.BestAskSize != nil && b.EMAAskSize != nil && b.EMAAskSize.IsPositive() {
		cur, _ := b.BestAskSize.Float64()
		ema, _ := b.EMAAskSize.Float64()
		if ema > 0 && cur < ema*(1-ratio) {
			askDrop = 1 - cur/ema
		}
	}
	return bidDrop, askDrop
}
