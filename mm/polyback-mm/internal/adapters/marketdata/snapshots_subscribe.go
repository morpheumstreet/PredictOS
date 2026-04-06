package marketdata

import (
	"context"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/polymarket/ws"
)

// SubscribeSnapshots emits a snapshot each time the CLOB client applies a full book update for an asset.
// The channel is not closed on ctx cancel (listeners are process-lifetime); stop consuming when ctx is done.
func SubscribeSnapshots(ctx context.Context, clob *polyws.ClobClient, mdp *WSProvider) <-chan domain.MarketSnapshot {
	out := make(chan domain.MarketSnapshot, 256)
	if clob == nil || mdp == nil {
		return out
	}
	clob.RegisterBookListener(func(assetID string) {
		if ctx.Err() != nil {
			return
		}
		snap, ok := mdp.Snapshot(ctx, assetID)
		if !ok {
			return
		}
		select {
		case out <- snap:
		default:
		}
	})
	return out
}
