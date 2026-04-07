package wiring

import (
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/kalshidflow"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/limitless"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/predictfun"
)

// PredictFunFromHft builds a Predict.fun client from merged HFT config (use after config.Load for YAML + env).
func PredictFunFromHft(h *config.Hft) *predictfun.PredictFun {
	c := h.PredictFun
	return predictfun.NewPredictFun(c.BaseURL, c.APIKey, c.PrivateKey)
}

// KalshiDFlowFromHft builds a DFlow Kalshi client from merged HFT config.
func KalshiDFlowFromHft(h *config.Hft) *kalshidflow.KalshiDFlow {
	c := h.KalshiDFlow
	return kalshidflow.NewKalshiDFlow(c.BaseURL, c.APIKey, c.EventTicker)
}

// LimitlessFromHft builds a Limitless client from merged HFT config.
func LimitlessFromHft(h *config.Hft) *limitless.Limitless {
	c := h.Limitless
	return limitless.NewLimitless(c.BaseURL, c.APIKey, c.WalletAddress)
}
