package gabagool

import (
	"time"

	"github.com/shopspring/decimal"
)

type Direction string

const (
	DirUp   Direction = "UP"
	DirDown Direction = "DOWN"
)

type Market struct {
	Slug       string
	UpTokenID  string
	DownTokenID string
	EndTime    time.Time
	MarketType string
}

type MarketInventory struct {
	UpShares         decimal.Decimal
	DownShares       decimal.Decimal
	LastUpFillAt     *time.Time
	LastDownFillAt   *time.Time
	LastUpFillPrice  *decimal.Decimal
	LastDownFillPrice *decimal.Decimal
	LastTopUpAt      *time.Time
}

func EmptyInventory() MarketInventory {
	return MarketInventory{UpShares: decimal.Zero, DownShares: decimal.Zero}
}

func (m MarketInventory) Imbalance() decimal.Decimal {
	return m.UpShares.Sub(m.DownShares)
}

func (m MarketInventory) AddUp(shares decimal.Decimal, at time.Time, price *decimal.Decimal) MarketInventory {
	m.UpShares = m.UpShares.Add(shares)
	m.LastUpFillAt = &at
	m.LastUpFillPrice = price
	return m
}

func (m MarketInventory) AddDown(shares decimal.Decimal, at time.Time, price *decimal.Decimal) MarketInventory {
	m.DownShares = m.DownShares.Add(shares)
	m.LastDownFillAt = &at
	m.LastDownFillPrice = price
	return m
}

func (m MarketInventory) WithTopUp(at time.Time) MarketInventory {
	m.LastTopUpAt = &at
	return m
}

type OrderState struct {
	OrderID            string
	Market             *Market
	TokenID            string
	Direction          Direction
	Price              decimal.Decimal
	Size               decimal.Decimal
	PlacedAt           time.Time
	MatchedSize        decimal.Decimal
	LastStatusCheckAt  *time.Time
	SecondsToEndAtEntry int64
}

type PlaceReason string

const (
	PlaceQuote    PlaceReason = "QUOTE"
	PlaceReplace  PlaceReason = "REPLACE"
	PlaceTaker    PlaceReason = "TAKER"
	PlaceTopUp    PlaceReason = "TOP_UP"
	PlaceFastTop  PlaceReason = "FAST_TOP_UP"
)

type CancelReason string

const (
	CancelShutdown        CancelReason = "SHUTDOWN"
	CancelOutsideLifetime CancelReason = "OUTSIDE_LIFETIME"
	CancelOutsideWindow   CancelReason = "OUTSIDE_TIME_WINDOW"
	CancelBookStale       CancelReason = "BOOK_STALE"
	CancelInsufficientEdge CancelReason = "INSUFFICIENT_EDGE"
	CancelReplacePrice    CancelReason = "REPLACE_PRICE"
	CancelReplaceSize     CancelReason = "REPLACE_SIZE"
	CancelReplaceBoth     CancelReason = "REPLACE_PRICE_AND_SIZE"
	CancelStaleTimeout    CancelReason = "STALE_TIMEOUT"
)

type ReplaceDecision int

const (
	ReplaceSkip ReplaceDecision = iota
	ReplacePlace
	ReplaceDo
)
