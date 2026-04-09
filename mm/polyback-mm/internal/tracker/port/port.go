// Package port defines tracker boundaries (dependency inversion): HTTP handlers and the app
// service depend on these interfaces; infrastructure implements them.
package port

import "context"

// PolymarketData is Polymarket's public data-api (positions + activity).
type PolymarketData interface {
	Positions(ctx context.Context, user string) ([]map[string]any, error)
	Activity(ctx context.Context, user string) ([]map[string]any, error)
}

// GammaMarket fetches raw market JSON by slug (Gamma REST).
type GammaMarket interface {
	MarketBySlug(slug string) ([]byte, error)
}
