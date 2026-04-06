package risk

import (
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/shopspring/decimal"
)

// OrderNotionalAllowed returns true when max_order_notional_usd is unset (0) or notional is within limit.
func OrderNotionalAllowed(root *config.Root, notional decimal.Decimal) bool {
	if root == nil {
		return false
	}
	mx := root.Hft.Risk.MaxOrderNotionalUsd
	if mx <= 0 {
		return true
	}
	return !notional.GreaterThan(decimal.NewFromFloat(mx))
}
