package executor

import (
	"context"
	"errors"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/api"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/ports/output"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/executorclient"
	"github.com/shopspring/decimal"
)

// PolymarketExecutor adapts executor HTTP client to the outbound port (limit bids for MM).
type PolymarketExecutor struct {
	client *executorclient.Client
}

var _ output.PolymarketExecutor = (*PolymarketExecutor)(nil)

func NewPolymarketExecutor(c *executorclient.Client) *PolymarketExecutor {
	if c == nil {
		return nil
	}
	return &PolymarketExecutor{client: c}
}

func ptrStr(s string) *string { return &s }

// PlaceLimitBid posts a GTC buy at price/size for token assetID.
func (p *PolymarketExecutor) PlaceLimitBid(ctx context.Context, assetID string, price, shares decimal.Decimal) error {
	if p == nil || p.client == nil {
		return errors.New("executor: nil polymarket executor client")
	}
	_ = ctx
	req := &api.LimitOrderRequest{
		TokenID: assetID, Side: domain.SideBuy, Price: price, Size: shares,
		OrderType: ptrStr("GTC"),
	}
	_, err := p.client.PlaceLimitOrder(req)
	return err
}
