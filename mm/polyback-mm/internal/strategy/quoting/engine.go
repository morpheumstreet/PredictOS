package quoting

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"math"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/ports/input"
	"github.com/shopspring/decimal"
)

// Engine implements ports/input.QuotingEngine (study.md-style bid/ask construction).
type Engine struct {
	cfg config.MarketMakerCfg
	vol input.VolatilitySpread // optional; nil skips dynamic EWMA spread
}

var _ input.QuotingEngine = (*Engine)(nil)

func NewEngine(cfg config.MarketMakerCfg, vol input.VolatilitySpread) *Engine {
	return &Engine{cfg: cfg, vol: vol}
}

func (e *Engine) GenerateQuote(snapshot *domain.MarketSnapshot, pos *domain.MMPosition, tox domain.ToxicitySignal, tickSize decimal.Decimal) (domain.MMQuote, error) {
	if snapshot == nil {
		return domain.MMQuote{}, errors.New("nil snapshot")
	}
	b := snapshot.Book
	if b.BestBid == nil || b.BestAsk == nil {
		return domain.MMQuote{}, errors.New("incomplete book")
	}
	fair := midFromBook(&b)
	if !fair.IsPositive() {
		return domain.MMQuote{}, errors.New("nonpositive fair")
	}
	skewTicks := 0
	if pos != nil {
		skewTicks = pos.SkewTicks
	}

	half := e.halfSpread(snapshot.AssetID, fair)
	bid := fair.Sub(half)
	ask := fair.Add(half)

	skew := decimal.NewFromInt(int64(skewTicks)).Mul(tickSize)
	bid = bid.Add(skew)
	ask = ask.Add(skew)

	imb := imbalanceSkew(&b, e.imbalanceScale())
	bid = bid.Add(imb)
	ask = ask.Add(imb)

	bid = bid.Sub(tox.BidPenalty)
	ask = ask.Add(tox.AskPenalty)

	nz := e.noise(tickSize)
	bid = bid.Add(nz)
	ask = ask.Add(nz)

	spread := ask.Sub(bid)
	if !spread.IsPositive() {
		if !tickSize.IsPositive() {
			tickSize = decimal.NewFromFloat(0.01)
		}
		spread = tickSize
		ask = bid.Add(spread)
	}
	return domain.MMQuote{Fair: fair, Spread: spread, Bid: bid, Ask: ask}, nil
}

// halfSpread is half of (base_spread + vol_spread_bonus + dynamic EWMA addon).
func (e *Engine) halfSpread(assetID string, fair decimal.Decimal) decimal.Decimal {
	base := e.cfg.BaseSpread
	if base <= 0 {
		base = 0.04
	}
	vb := e.cfg.VolSpreadBonus
	if vb < 0 {
		vb = 0
	}
	total := decimal.NewFromFloat(base + vb)
	if e.vol != nil {
		total = total.Add(e.vol.SpreadAddon(assetID, fair))
	}
	return total.Div(decimal.NewFromInt(2))
}

func (e *Engine) imbalanceScale() float64 {
	s := e.cfg.ImbalanceSkewScale
	if s <= 0 {
		return 0.02
	}
	return s
}

func (e *Engine) noise(tickSize decimal.Decimal) decimal.Decimal {
	sigma := e.cfg.NoiseSigma
	if sigma <= 0 {
		return decimal.Zero
	}
	maxT := e.cfg.NoiseMaxTicks
	if maxT <= 0 {
		maxT = 1
	}
	if !tickSize.IsPositive() {
		tickSize = decimal.NewFromFloat(0.01)
	}
	z := gaussianRand()
	w := decimal.NewFromFloat(z * sigma)
	maxW := tickSize.Mul(decimal.NewFromInt(int64(maxT)))
	if w.Abs().GreaterThan(maxW) {
		if w.IsNegative() {
			w = maxW.Neg()
		} else {
			w = maxW
		}
	}
	return w
}

func gaussianRand() float64 {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return 0
	}
	u1 := binary.BigEndian.Uint64(buf[:8])
	u2 := binary.BigEndian.Uint64(buf[8:])
	f1 := float64(u1) / float64(^uint64(0))
	f1 = math.Max(1e-15, math.Min(f1, 1-1e-15))
	f2 := float64(u2) / float64(^uint64(0))
	return math.Sqrt(-2*math.Log(f1)) * math.Cos(2*math.Pi*f2)
}
