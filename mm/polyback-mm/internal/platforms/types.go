package platforms

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

// BookLevel is one price level in an order book.
type BookLevel struct {
	Price decimal.Decimal
	Size  decimal.Decimal
}

// Orderbook is normalized to YES-outcome bids/asks where the venue exposes a single book;
// some venues only provide a YES book at the REST layer.
type Orderbook struct {
	Bids []BookLevel
	Asks []BookLevel
}

// Market is a normalized binary (YES/NO) market snapshot.
type Market struct {
	ID          string
	Slug        string
	Question    string
	Description string
	Category    string

	YesPrice decimal.Decimal
	NoPrice  decimal.Decimal

	YesTokenID string
	NoTokenID  string

	Volume24h decimal.Decimal
	Liquidity decimal.Decimal

	CreatedAt *time.Time
	ExpiresAt *time.Time

	Active   bool
	Resolved bool

	Raw json.RawMessage
}

// Order is a normalized open or historical order.
type Order struct {
	ID             string
	MarketID       string
	Side           string // "yes" / "no"
	Size           decimal.Decimal
	Price          decimal.Decimal
	OrderType      string
	Status         string
	CreatedAt      *time.Time
	FilledSize     decimal.Decimal
	RemainingSize  decimal.Decimal
	Raw            json.RawMessage
}

// Position is a normalized outcome position.
type Position struct {
	MarketID     string
	Side         string
	Size         decimal.Decimal
	AvgPrice     decimal.Decimal
	CurrentPrice decimal.Decimal
	PnL          decimal.Decimal
}

// PlaceOrderRequest is input for venue-native limit buys.
type PlaceOrderRequest struct {
	MarketID string
	Side     string // "yes" / "no"
	Size     decimal.Decimal
	Price    decimal.Decimal
	Type     string // "limit" default
}

// OrderResult is returned after SendOrder.
type OrderResult struct {
	Success bool
	OrderID string
	Status  string
	Raw     json.RawMessage
	Error   string
}
