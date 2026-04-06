package domain

import "github.com/shopspring/decimal"

// ToxicityLevel is a coarse classification from heuristics.
type ToxicityLevel int

const (
	ToxicityNone ToxicityLevel = iota
	ToxicityElevated
	ToxicityHigh
)

// ToxicitySignal drives spread/price adjustments. Penalties are non-negative price deltas
// applied as: bid -= BidPenalty, ask += AskPenalty (widen around fair).
type ToxicitySignal struct {
	Level            ToxicityLevel
	BurstTrades      int
	ImpactScore      float64
	LiquidityDropBid float64
	LiquidityDropAsk float64
	BidPenalty       decimal.Decimal
	AskPenalty       decimal.Decimal
	Unsafe           bool
	// PauseBidQuotes / PauseAskQuotes: depth collapse vs EMA; maker should not quote that side.
	PauseBidQuotes bool
	PauseAskQuotes bool
}
