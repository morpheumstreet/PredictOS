package quoting

import (
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/shopspring/decimal"
)

func midFromBook(b *domain.OrderBookL2) decimal.Decimal {
	if b == nil || b.BestBid == nil || b.BestAsk == nil {
		return decimal.Zero
	}
	return b.BestBid.Add(*b.BestAsk).Div(decimal.NewFromInt(2))
}

// imbalanceSkew maps top-of-book size imbalance to a small price shift (same sign on bid and ask per study.md).
func imbalanceSkew(b *domain.OrderBookL2, scale float64) decimal.Decimal {
	if b == nil || scale <= 0 {
		return decimal.Zero
	}
	if b.BestBidSize == nil || b.BestAskSize == nil {
		return decimal.Zero
	}
	bb, ba := *b.BestBidSize, *b.BestAskSize
	if !bb.IsPositive() && !ba.IsPositive() {
		return decimal.Zero
	}
	sum := bb.Add(ba)
	if !sum.IsPositive() {
		return decimal.Zero
	}
	r := bb.Sub(ba).Div(sum)
	f, _ := r.Float64()
	return decimal.NewFromFloat(f * scale)
}
