package ports

import (
	"encoding/json"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/api"
)

// OrderSimulator abstracts paper vs live order execution (DIP). HTTP layer depends on this port.
type OrderSimulator interface {
	Enabled() bool
	PlaceLimitOrder(req *api.LimitOrderRequest) *api.OrderSubmissionResult
	PlaceMarketOrder(req *api.MarketOrderRequest) *api.OrderSubmissionResult
	CancelOrder(orderID string) json.RawMessage
	GetOrder(orderID string) json.RawMessage
	GetPositions(limit, offset int) []api.PolymarketPosition
}
