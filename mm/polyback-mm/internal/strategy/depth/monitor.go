package depth

import (
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
)

// Pauses returns whether to stop quoting the bid side and/or ask side when top size
// has fallen vs EMA baseline beyond the configured drop ratio (same idea as toxicity liquidity drop).
func Pauses(book *domain.OrderBookL2, cfg config.MarketMakerCfg) (pauseBid, pauseAsk bool) {
	if book == nil || !cfg.DepthPauseEnabled {
		return false, false
	}
	ratio := cfg.DepthPauseDropRatio
	if ratio <= 0 {
		ratio = cfg.LiquidityDropRatio
	}
	if ratio <= 0 {
		ratio = 0.35
	}
	bidDrop, askDrop := liquidityDrops(book, ratio)
	return bidDrop > 0, askDrop > 0
}

func liquidityDrops(b *domain.OrderBookL2, ratio float64) (bidDrop, askDrop float64) {
	if b == nil || ratio <= 0 || ratio >= 1 {
		return 0, 0
	}
	if b.BestBidSize != nil && b.EMABidSize != nil && b.EMABidSize.IsPositive() {
		cur, _ := b.BestBidSize.Float64()
		ema, _ := b.EMABidSize.Float64()
		if ema > 0 && cur < ema*(1-ratio) {
			bidDrop = 1 - cur/ema
		}
	}
	if b.BestAskSize != nil && b.EMAAskSize != nil && b.EMAAskSize.IsPositive() {
		cur, _ := b.BestAskSize.Float64()
		ema, _ := b.EMAAskSize.Float64()
		if ema > 0 && cur < ema*(1-ratio) {
			askDrop = 1 - cur/ema
		}
	}
	return bidDrop, askDrop
}
