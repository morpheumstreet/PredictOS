package position

import (
	"math"
	"sort"
	"strings"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/jsonnum"
)

func filterTradesForMarket(activities []map[string]any, conditionID, upTok, downTok string) []map[string]any {
	var out []map[string]any
	for _, a := range activities {
		if strings.ToUpper(strings.TrimSpace(strVal(a["type"]))) != "TRADE" {
			continue
		}
		asset, _ := a["asset"].(string)
		if asset != upTok && asset != downTok {
			continue
		}
		if conditionID != "" {
			cid, _ := a["conditionId"].(string)
			if strings.TrimSpace(cid) != conditionID {
				continue
			}
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		return jsonnum.AsFloat64(out[i]["timestamp"]) < jsonnum.AsFloat64(out[j]["timestamp"])
	})
	return out
}

func strVal(v any) string {
	s, _ := v.(string)
	return s
}

func countTradesByToken(trades []map[string]any, upTok, downTok string) (yesCount, noCount int) {
	for _, t := range trades {
		asset, _ := t["asset"].(string)
		switch asset {
		case upTok:
			yesCount++
		case downTok:
			noCount++
		}
	}
	return yesCount, noCount
}

func avgFromTrades(trades []map[string]any, token string, currentSize float64) (avgPrice, estCost float64) {
	var sh, cost float64
	for _, t := range trades {
		asset, _ := t["asset"].(string)
		if asset != token {
			continue
		}
		side := strings.ToUpper(strings.TrimSpace(strVal(t["side"])))
		sz := jsonnum.AsFloat64(t["size"])
		price := jsonnum.AsFloat64(t["price"])
		usdc := jsonnum.AsFloat64(t["usdcSize"])
		if usdc < 1e-12 && sz > 0 && price > 0 {
			usdc = sz * price
		}
		switch side {
		case "BUY":
			sh += sz
			cost += usdc
		case "SELL":
			if sh < 1e-12 {
				continue
			}
			avg := cost / sh
			cost -= math.Min(sz, sh) * avg
			sh = math.Max(0, sh-sz)
		}
	}
	if currentSize > 1e-9 && sh > 1e-9 {
		avgPrice = cost / sh
		estCost = (cost / sh) * currentSize
		return avgPrice, estCost
	}
	if sh > 1e-9 {
		avgPrice = cost / sh
		estCost = cost
	}
	return avgPrice, estCost
}

func inventoryFromTrades(trades []map[string]any, upTok, downTok string) (yesSz, noSz, yesAvg, noAvg, yesCost, noCost float64) {
	var ySh, yC, nSh, nC float64
	for _, t := range trades {
		asset, _ := t["asset"].(string)
		side := strings.ToUpper(strings.TrimSpace(strVal(t["side"])))
		sz := jsonnum.AsFloat64(t["size"])
		price := jsonnum.AsFloat64(t["price"])
		usdc := jsonnum.AsFloat64(t["usdcSize"])
		if usdc < 1e-12 && sz > 0 && price > 0 {
			usdc = sz * price
		}
		var sh, cost *float64
		switch asset {
		case upTok:
			sh, cost = &ySh, &yC
		case downTok:
			sh, cost = &nSh, &nC
		default:
			continue
		}
		switch side {
		case "BUY":
			*sh += sz
			*cost += usdc
		case "SELL":
			if *sh < 1e-12 {
				continue
			}
			avg := *cost / *sh
			take := math.Min(sz, *sh)
			*cost -= take * avg
			*sh -= take
		}
	}
	if ySh > 1e-9 {
		yesSz, yesAvg, yesCost = ySh, yC/ySh, yC
	}
	if nSh > 1e-9 {
		noSz, noAvg, noCost = nSh, nC/nSh, nC
	}
	return yesSz, noSz, yesAvg, noAvg, yesCost, noCost
}
