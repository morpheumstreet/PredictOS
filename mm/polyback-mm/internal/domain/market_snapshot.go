package domain

import "time"

// MarketSnapshot is a point-in-time view for quoting and toxicity.
type MarketSnapshot struct {
	AssetID    string
	Book       OrderBookL2
	Trades     []Trade
	ObservedAt time.Time
}
