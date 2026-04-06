package ws

import (
	"time"

	"github.com/shopspring/decimal"
)

// BookLevel is one price level from the CLOB book message (top-N retained).
type BookLevel struct {
	Price *decimal.Decimal `json:"price,omitempty"`
	Size  *decimal.Decimal `json:"size,omitempty"`
}

type TopOfBook struct {
	BestBid        *decimal.Decimal `json:"bestBid,omitempty"`
	BestAsk        *decimal.Decimal `json:"bestAsk,omitempty"`
	BestBidSize    *decimal.Decimal `json:"bestBidSize,omitempty"`
	BestAskSize    *decimal.Decimal `json:"bestAskSize,omitempty"`
	LastTradePrice *decimal.Decimal `json:"lastTradePrice,omitempty"`
	UpdatedAt      *time.Time       `json:"updatedAt,omitempty"`
	LastTradeAt    *time.Time       `json:"lastTradeAt,omitempty"`
	BidLevels      []BookLevel      `json:"bidLevels,omitempty"`
	AskLevels      []BookLevel      `json:"askLevels,omitempty"`
}
