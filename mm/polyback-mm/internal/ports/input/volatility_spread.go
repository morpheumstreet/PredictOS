package input

import "github.com/shopspring/decimal"

// VolatilitySpread maps recent mid volatility into extra spread width (probability units, same as BaseSpread).
type VolatilitySpread interface {
	SpreadAddon(assetID string, mid decimal.Decimal) decimal.Decimal
}
