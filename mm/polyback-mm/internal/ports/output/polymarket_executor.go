package output

import (
	"context"

	"github.com/shopspring/decimal"
)

// PolymarketExecutor is the outbound order surface (adapter to executor HTTP in a later slice).
type PolymarketExecutor interface {
	PlaceLimitBid(ctx context.Context, assetID string, price, shares decimal.Decimal) error
}
