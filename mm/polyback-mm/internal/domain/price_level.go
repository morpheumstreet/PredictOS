package domain

import "github.com/shopspring/decimal"

// PriceLevel is a full-book price/size level (used when WS sends L2 arrays).
type PriceLevel struct {
	Price decimal.Decimal
	Size  decimal.Decimal
}
