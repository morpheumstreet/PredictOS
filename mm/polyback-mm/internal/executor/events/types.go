package events

import (
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/shopspring/decimal"
)

type ExecutorOrderStatus struct {
	OrderID        string           `json:"orderId"`
	TokenID        string           `json:"tokenId"`
	Side           domain.OrderSide `json:"side"`
	RequestedPrice *decimal.Decimal `json:"requestedPrice,omitempty"`
	RequestedSize  *decimal.Decimal `json:"requestedSize,omitempty"`
	Status         string           `json:"status"`
	Matched        *decimal.Decimal `json:"matched,omitempty"`
	Remaining      *decimal.Decimal `json:"remaining,omitempty"`
	OrderJSON      string           `json:"orderJson,omitempty"`
	Error          string           `json:"error,omitempty"`
}

type ExecutorLimitOrder struct {
	TokenID   string           `json:"tokenId"`
	Side      domain.OrderSide `json:"side"`
	Price     decimal.Decimal  `json:"price"`
	Size      decimal.Decimal  `json:"size"`
	Mode      string           `json:"mode,omitempty"`
	OrderID   string           `json:"orderId,omitempty"`
	Error     string           `json:"error,omitempty"`
}
