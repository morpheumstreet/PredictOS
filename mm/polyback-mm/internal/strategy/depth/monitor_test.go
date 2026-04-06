package depth

import (
	"testing"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/shopspring/decimal"
)

func TestPauses_disabled(t *testing.T) {
	cfg := config.MarketMakerCfg{DepthPauseEnabled: false, DepthPauseDropRatio: 0.3}
	b := &domain.OrderBookL2{
		BestBidSize: dec("2"), BestAskSize: dec("10"),
		EMABidSize: dec("20"), EMAAskSize: dec("10"),
	}
	pb, pa := Pauses(b, cfg)
	if pb || pa {
		t.Fatalf("want no pause when disabled")
	}
}

func TestPauses_bidSideThin(t *testing.T) {
	cfg := config.MarketMakerCfg{
		DepthPauseEnabled:   true,
		DepthPauseDropRatio: 0.3,
	}
	b := &domain.OrderBookL2{
		BestBidSize: dec("2"), BestAskSize: dec("10"),
		EMABidSize: dec("20"), EMAAskSize: dec("10"),
	}
	pb, pa := Pauses(b, cfg)
	if !pb {
		t.Fatalf("want bid pause when bid liquidity dropped")
	}
	if pa {
		t.Fatalf("ask should not pause in this fixture")
	}
}

func dec(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}
