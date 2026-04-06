package toxicity

import (
	"testing"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/shopspring/decimal"
)

func decp(s string) *decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return &d
}

func TestDetector_burstElevated(t *testing.T) {
	cfg := config.MarketMakerCfg{
		TradeWindowMillis:   5000,
		BurstTradeCount:     3,
		ToxicityUnsafeBurst: 20,
	}
	d := NewDetector(cfg)
	now := time.Now().UTC()
	trades := []domain.Trade{
		{Price: *decp("0.5"), Timestamp: now.Add(-10 * time.Millisecond)},
		{Price: *decp("0.5"), Timestamp: now.Add(-20 * time.Millisecond)},
		{Price: *decp("0.5"), Timestamp: now.Add(-30 * time.Millisecond)},
	}
	bb, ba := decp("0.48"), decp("0.52")
	b := &domain.OrderBookL2{
		BestBid: bb, BestAsk: ba,
		BestBidSize: decp("10"), BestAskSize: decp("10"),
	}
	sig := d.Assess(trades, b)
	if sig.Level < domain.ToxicityElevated {
		t.Fatalf("expected elevated+, got %v burst=%d", sig.Level, sig.BurstTrades)
	}
	if !sig.BidPenalty.IsPositive() {
		t.Fatalf("expected positive penalty, got %v", sig.BidPenalty)
	}
}

func TestDetector_unsafeBurst(t *testing.T) {
	cfg := config.MarketMakerCfg{
		TradeWindowMillis:   5000,
		BurstTradeCount:     2,
		ToxicityUnsafeBurst: 4,
	}
	d := NewDetector(cfg)
	now := time.Now().UTC()
	var trades []domain.Trade
	for i := 0; i < 5; i++ {
		trades = append(trades, domain.Trade{Price: *decp("0.5"), Timestamp: now.Add(time.Duration(-i) * time.Millisecond)})
	}
	bb, ba := decp("0.49"), decp("0.51")
	b := &domain.OrderBookL2{BestBid: bb, BestAsk: ba}
	sig := d.Assess(trades, b)
	if !sig.Unsafe {
		t.Fatalf("expected unsafe")
	}
}

func TestDetector_liquidityDrop(t *testing.T) {
	cfg := config.MarketMakerCfg{
		LiquidityDropRatio: 0.3,
		ToxicityPenaltyMax: 0.02,
	}
	d := NewDetector(cfg)
	b := &domain.OrderBookL2{
		BestBid: decp("0.4"), BestAsk: decp("0.6"),
		BestBidSize: decp("2"), BestAskSize: decp("10"),
		EMABidSize: decp("20"), EMAAskSize: decp("10"),
	}
	sig := d.Assess(nil, b)
	if sig.LiquidityDropBid <= 0 {
		t.Fatalf("expected bid-side drop signal, got %+v", sig)
	}
	if sig.Level < domain.ToxicityElevated {
		t.Fatalf("expected elevated from liquidity drop, got %v", sig.Level)
	}
}
