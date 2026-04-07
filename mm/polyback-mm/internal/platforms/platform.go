package platforms

import (
	"context"

	"github.com/shopspring/decimal"
)

// Platform is the shared surface for prediction-market venues (aligned with cryptomaid base.PredictionMarket).
type Platform interface {
	Name() string

	GetAllMarkets(ctx context.Context) ([]Market, error)
	GetMarket(ctx context.Context, marketID string) (*Market, error)
	GetPrices(ctx context.Context, marketID string) (yes, no decimal.Decimal, err error)
	GetOrderbook(ctx context.Context, marketID string) (Orderbook, error)

	// SendOrder submits a limit order using venue-native signing / REST (where supported).
	SendOrder(ctx context.Context, req PlaceOrderRequest) (*OrderResult, error)
	CancelOrder(ctx context.Context, orderID string) error
	ListOrders(ctx context.Context, marketID *string) ([]Order, error)
	GetPositions(ctx context.Context) ([]Position, error)
	// GetBalance returns venue-specific keys (e.g. USDC, points, locked).
	GetBalance(ctx context.Context) (map[string]decimal.Decimal, error)

	HealthCheck(ctx context.Context) error
}
