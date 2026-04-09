package position

import (
	"strings"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/tracker/jsonnum"
)

func positionSidesFromAPI(positions []map[string]any, upTok, downTok string) (yesSz, noSz, yesAvg, noAvg, yesCost, noCost float64, conditionID string) {
	for _, p := range positions {
		aid, _ := p["asset"].(string)
		if aid != upTok && aid != downTok {
			continue
		}
		sz := jsonnum.AsFloat64(p["size"])
		avg := jsonnum.AsFloat64(p["avgPrice"])
		initV := jsonnum.AsFloat64(p["initialValue"])
		usdc := jsonnum.AsFloat64(p["usdcSize"])
		if cid, ok := p["conditionId"].(string); ok && strings.TrimSpace(cid) != "" {
			conditionID = strings.TrimSpace(cid)
		}
		switch aid {
		case upTok:
			yesSz = sz
			yesAvg = avg
			yesCost = initV
			if yesCost < 1e-12 && usdc > 0 {
				yesCost = usdc
			}
			if yesCost < 1e-12 && yesAvg > 0 && yesSz > 0 {
				yesCost = yesAvg * yesSz
			}
		case downTok:
			noSz = sz
			noAvg = avg
			noCost = initV
			if noCost < 1e-12 && usdc > 0 {
				noCost = usdc
			}
			if noCost < 1e-12 && noAvg > 0 && noSz > 0 {
				noCost = noAvg * noSz
			}
		}
	}
	return yesSz, noSz, yesAvg, noAvg, yesCost, noCost, conditionID
}
