package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// OrderBookL2 v1 mirrors top-of-book; deeper levels can be added later.
type OrderBookL2 struct {
	AssetID        string
	BestBid        *decimal.Decimal
	BestAsk        *decimal.Decimal
	BestBidSize    *decimal.Decimal
	BestAskSize    *decimal.Decimal
	UpdatedAt      *time.Time
	LastTradeAt    *time.Time
	LastTradePrice *decimal.Decimal
	// EMA baselines for size (from feed); used for liquidity-drop toxicity.
	EMABidSize *decimal.Decimal
	EMAAskSize *decimal.Decimal
	// Optional top-of-book depth from WS (see polymarket/ws TopOfBook.BidLevels).
	BidLevels []PriceLevel
	AskLevels []PriceLevel
}
