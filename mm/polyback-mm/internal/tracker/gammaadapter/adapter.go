package gammaadapter

import (
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/gamma"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/port"
)

// FromClient wraps the shared Gamma HTTP client as port.GammaMarket.
type FromClient struct {
	Inner *gamma.Client
}

var _ port.GammaMarket = (*FromClient)(nil)

// MarketBySlug implements port.GammaMarket.
func (a *FromClient) MarketBySlug(slug string) ([]byte, error) {
	return a.Inner.MarketBySlug(slug)
}
