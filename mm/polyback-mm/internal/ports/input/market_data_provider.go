package input

import (
	"context"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
)

// MarketDataProvider returns snapshots for an asset (pull-based v1).
type MarketDataProvider interface {
	Snapshot(ctx context.Context, assetID string) (domain.MarketSnapshot, bool)
}
