package position

import "math"

// PairMetrics summarizes YES/NO legs for the terminal dashboard.
func PairMetrics(yesShares, noShares, avgYes, avgNo, yesCost, noCost float64) (
	status string, pairCost any, minShares, guaranteedPayout, totalCost, guaranteedProfit, returnPercent float64,
) {
	const eps = 1e-6
	switch {
	case yesShares < eps && noShares < eps:
		return "NO_POSITION", nil, 0, 0, 0, 0, 0
	case yesShares >= eps && noShares < eps:
		return "DIRECTIONAL_YES", nil, 0, 0, 0, 0, 0
	case noShares >= eps && yesShares < eps:
		return "DIRECTIONAL_NO", nil, 0, 0, 0, 0, 0
	}
	if avgYes < eps && yesCost > eps && yesShares > eps {
		avgYes = yesCost / yesShares
	}
	if avgNo < eps && noCost > eps && noShares > eps {
		avgNo = noCost / noShares
	}
	pc := avgYes + avgNo
	minShares = math.Min(yesShares, noShares)
	totalCost = minShares * pc
	guaranteedPayout = minShares * 1.0
	guaranteedProfit = guaranteedPayout - totalCost
	if totalCost > 1e-12 {
		returnPercent = 100.0 * guaranteedProfit / totalCost
	}
	st := "LOSS_RISK"
	if pc < 1-eps {
		st = "PROFIT_LOCKED"
	} else if pc <= 1+eps {
		st = "BREAK_EVEN"
	}
	return st, pc, minShares, guaranteedPayout, totalCost, guaranteedProfit, returnPercent
}

func sideSnapshot(shares, costUsd, avgPrice float64, tradeCount int) map[string]any {
	ap := avgPrice
	if shares > 1e-9 && costUsd > 1e-9 && ap < 1e-12 {
		ap = costUsd / shares
	}
	return map[string]any{
		"shares":        shares,
		"costUsd":       costUsd,
		"avgPrice":      ap,
		"ordersPlaced":  tradeCount,
		"ordersFilled":  tradeCount,
		"pendingShares": 0.0,
	}
}
