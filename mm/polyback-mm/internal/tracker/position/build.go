package position

import (
	"context"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/market"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/port"
)

// BuildSnapshot loads data-api state for one wallet and one resolved market (terminal JSON shape).
func BuildSnapshot(ctx context.Context, data port.PolymarketData, wallet string, m market.UpDown) (map[string]any, error) {
	cid := m.ConditionID
	positions, err := data.Positions(ctx, wallet)
	if err != nil {
		return nil, err
	}

	yesShares, noShares, yesAvg, noAvg, yesCost, noCost, posCond := positionSidesFromAPI(positions, m.UpToken, m.DownToken)
	if cid == "" {
		cid = posCond
	}

	activities, err := data.Activity(ctx, wallet)
	if err != nil {
		return nil, err
	}
	trades := filterTradesForMarket(activities, cid, m.UpToken, m.DownToken)
	yesOrders, noOrders := countTradesByToken(trades, m.UpToken, m.DownToken)

	if yesShares < 1e-9 && noShares < 1e-9 {
		y2, n2, ay, ny, cy, nc2 := inventoryFromTrades(trades, m.UpToken, m.DownToken)
		if y2 > 1e-9 || n2 > 1e-9 {
			yesShares, noShares = y2, n2
			yesAvg, noAvg = ay, ny
			yesCost, noCost = cy, nc2
		}
	} else {
		if yesAvg < 1e-12 && yesShares > 1e-9 {
			yesAvg, yesCost = avgFromTrades(trades, m.UpToken, yesShares)
		}
		if noAvg < 1e-12 && noShares > 1e-9 {
			noAvg, noCost = avgFromTrades(trades, m.DownToken, noShares)
		}
		if yesCost < 1e-12 && yesShares > 1e-9 {
			yesCost = yesAvg * yesShares
		}
		if noCost < 1e-12 && noShares > 1e-9 {
			noCost = noAvg * noShares
		}
	}

	status, pairCost, minShares, gpayout, totalCost, gprofit, retPct := PairMetrics(yesShares, noShares, yesAvg, noAvg, yesCost, noCost)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	return map[string]any{
		"walletAddress":    wallet,
		"marketSlug":       m.Slug,
		"marketTitle":      m.Title,
		"tokenIds":         map[string]any{"up": m.UpToken, "down": m.DownToken},
		"yes":              sideSnapshot(yesShares, yesCost, yesAvg, yesOrders),
		"no":               sideSnapshot(noShares, noCost, noAvg, noOrders),
		"pairCost":         pairCost,
		"status":           status,
		"minShares":        minShares,
		"guaranteedPayout": gpayout,
		"totalCost":        totalCost,
		"guaranteedProfit": gprofit,
		"returnPercent":    retPct,
		"lastUpdated":      now,
	}, nil
}
