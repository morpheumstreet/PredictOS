package toxicity

import (
	"math"
	"strings"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
)

// VPINUnsafe estimates volume-synchronized imbalance from recent trades when Side and Size are present.
// Returns (imbalance in [0,1], true if threshold exceeded). Degrades gracefully when sides are missing.
func VPINUnsafe(trades []domain.Trade, minTrades int, threshold float64) (imbalance float64, unsafe bool) {
	if threshold <= 0 || threshold > 1 || minTrades <= 0 {
		return 0, false
	}
	var buyVol, sellVol float64
	nTagged := 0
	for _, t := range trades {
		if t.Size == nil {
			continue
		}
		sz, _ := t.Size.Float64()
		if sz <= 0 {
			continue
		}
		side := strings.ToUpper(strings.TrimSpace(t.Side))
		switch side {
		case "BUY", "B", "YES":
			buyVol += sz
			nTagged++
		case "SELL", "S", "NO":
			sellVol += sz
			nTagged++
		default:
			continue
		}
	}
	tot := buyVol + sellVol
	if tot < 1e-12 || nTagged < minTrades {
		return 0, false
	}
	imb := math.Abs(buyVol-sellVol) / tot
	return imb, imb >= threshold
}
