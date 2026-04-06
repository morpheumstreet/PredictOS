package domain

import "github.com/shopspring/decimal"

// MMPosition is inventory context for one market leg (e.g. one outcome token).
// ImbalanceShares follows gabagool: positive = long UP-like leg convention from caller.
type MMPosition struct {
	MarketSlug      string
	AssetID         string
	ImbalanceShares decimal.Decimal
	SkewTicks       int
}
