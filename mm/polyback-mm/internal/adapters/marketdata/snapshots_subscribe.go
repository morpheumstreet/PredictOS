package marketdata

import (
	"context"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/ws"
)

// SubscribeSnapshots is equivalent to (*WSProvider).SubscribeL2. The CLOB argument is unused;
// the provider must have been built with the same client you use for subscriptions.
// Prefer calling SubscribeL2 on WSProvider directly.
func SubscribeSnapshots(ctx context.Context, _ *polyws.ClobClient, mdp *WSProvider) <-chan domain.MarketSnapshot {
	if mdp == nil {
		return make(chan domain.MarketSnapshot)
	}
	ch, err := mdp.SubscribeL2(ctx)
	if err != nil {
		return make(chan domain.MarketSnapshot)
	}
	return ch
}
