package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Trade is a public trade or last-trade tick from the feed (size may be unknown).
type Trade struct {
	AssetID   string
	Price     decimal.Decimal
	Size      *decimal.Decimal
	Side      string
	Timestamp time.Time
}
