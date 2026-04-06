package quoting

import (
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/ports/input"
	"github.com/shopspring/decimal"
)

type constVolSpread struct{ add decimal.Decimal }

func (c *constVolSpread) SpreadAddon(string, decimal.Decimal) decimal.Decimal {
	if c == nil {
		return decimal.Zero
	}
	return c.add
}

var _ input.VolatilitySpread = (*constVolSpread)(nil)

func TestEngine_GenerateQuote_bidBelowAsk(t *testing.T) {
	cfg := config.MarketMakerCfg{
		BaseSpread:         0.04,
		NoiseSigma:         0,
		ImbalanceSkewScale: 0,
	}
	e := NewEngine(cfg, nil)
	tick := decimal.NewFromFloat(0.01)
	bb := decimal.RequireFromString("0.48")
	ba := decimal.RequireFromString("0.52")
	snap := &domain.MarketSnapshot{
		AssetID: "test-asset",
		Book: domain.OrderBookL2{
			BestBid:     &bb,
			BestAsk:     &ba,
			BestBidSize: ptrDec("100"),
			BestAskSize: ptrDec("100"),
		},
	}
	pos := &domain.MMPosition{SkewTicks: 0}
	tox := domain.ToxicitySignal{}
	q, err := e.GenerateQuote(snap, pos, tox, tick)
	if err != nil {
		t.Fatal(err)
	}
	if !q.Bid.LessThan(q.Ask) {
		t.Fatalf("bid >= ask: bid=%v ask=%v", q.Bid, q.Ask)
	}
	// fair 0.50, half spread 0.02 -> bid ~0.48 ask ~0.52 before tiny float
	if q.Fair.StringFixed(2) != "0.50" {
		t.Fatalf("fair want 0.50 got %v", q.Fair)
	}
}

func TestEngine_GenerateQuote_toxicityWidens(t *testing.T) {
	cfg := config.MarketMakerCfg{BaseSpread: 0.02, NoiseSigma: 0, ImbalanceSkewScale: 0}
	e := NewEngine(cfg, nil)
	tick := decimal.NewFromFloat(0.01)
	bb := decimal.RequireFromString("0.49")
	ba := decimal.RequireFromString("0.51")
	snap := &domain.MarketSnapshot{
		AssetID: "t1",
		Book:    domain.OrderBookL2{BestBid: &bb, BestAsk: &ba},
	}
	pos := &domain.MMPosition{}
	tox := domain.ToxicitySignal{
		BidPenalty: decimal.RequireFromString("0.01"),
		AskPenalty: decimal.RequireFromString("0.01"),
	}
	q, err := e.GenerateQuote(snap, pos, tox, tick)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := e.GenerateQuote(snap, pos, domain.ToxicitySignal{}, tick)
	if !q.Spread.GreaterThan(base.Spread) {
		t.Fatalf("expected wider spread with toxicity: %v vs %v", q.Spread, base.Spread)
	}
}

func ptrDec(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}

func TestEngine_GenerateQuote_volSpreadAddonWidens(t *testing.T) {
	cfg := config.MarketMakerCfg{BaseSpread: 0.04, NoiseSigma: 0, ImbalanceSkewScale: 0}
	e0 := NewEngine(cfg, nil)
	add := decimal.RequireFromString("0.02")
	e1 := NewEngine(cfg, &constVolSpread{add: add})
	tick := decimal.NewFromFloat(0.01)
	bb := decimal.RequireFromString("0.48")
	ba := decimal.RequireFromString("0.52")
	snap := &domain.MarketSnapshot{
		AssetID: "v1",
		Book: domain.OrderBookL2{
			BestBid: &bb, BestAsk: &ba,
		},
	}
	q0, err := e0.GenerateQuote(snap, nil, domain.ToxicitySignal{}, tick)
	if err != nil {
		t.Fatal(err)
	}
	q1, err := e1.GenerateQuote(snap, nil, domain.ToxicitySignal{}, tick)
	if err != nil {
		t.Fatal(err)
	}
	if !q1.Spread.GreaterThan(q0.Spread) {
		t.Fatalf("vol addon should widen spread: %v vs %v", q1.Spread, q0.Spread)
	}
}
