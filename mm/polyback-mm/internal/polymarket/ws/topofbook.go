package ws

import (
	"time"

	"github.com/shopspring/decimal"
)

type TopOfBook struct {
	BestBid       *decimal.Decimal `json:"bestBid,omitempty"`
	BestAsk       *decimal.Decimal `json:"bestAsk,omitempty"`
	BestBidSize   *decimal.Decimal `json:"bestBidSize,omitempty"`
	BestAskSize   *decimal.Decimal `json:"bestAskSize,omitempty"`
	LastTradePrice *decimal.Decimal `json:"lastTradePrice,omitempty"`
	UpdatedAt      *time.Time       `json:"updatedAt,omitempty"`
	LastTradeAt    *time.Time       `json:"lastTradeAt,omitempty"`
}
