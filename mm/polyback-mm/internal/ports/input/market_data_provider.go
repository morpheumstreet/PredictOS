package input

import (
	"context"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
)

// MarketDataProvider returns snapshots for an asset (pull-based) and optional push updates.
// SubscribeL2 emits a MarketSnapshot after each full book update for an asset (same fields as Snapshot).
type MarketDataProvider interface {
	Snapshot(ctx context.Context, assetID string) (domain.MarketSnapshot, bool)
	SubscribeL2(ctx context.Context) (<-chan domain.MarketSnapshot, error)
}
