package prompts

import (
	"encoding/json"
	"fmt"
)

// ArbitrageAnalysis asks the model for a full ArbitrageAnalysis-shaped JSON (contract aligned with arbitrage-finder).
func ArbitrageAnalysis(sourceLabel string, sourceMarkets []any, otherLabel string, otherMarkets []any, eventTitle string) (system, user string) {
	system = `You are an expert in cross-platform prediction market arbitrage (Polymarket vs Kalshi).
You compare markets, decide if they are the same underlying event, and assess lock/arb structure.
Output ONLY valid JSON matching the ArbitrageAnalysis schema with keys:
isSameMarket, sameMarketConfidence, marketComparisonReasoning,
polymarketData (optional object), kalshiData (optional object),
arbitrage (object with hasArbitrage, profitPercent optional, strategy optional with buyYesOn, buyYesPrice, buyNoOn, buyNoPrice, totalCost, guaranteedPayout, netProfit),
summary, risks (array), recommendation.
Use buyYesOn and buyNoOn as strings "polymarket" or "kalshi". Prices in 0-100 scale (cents probability).`

	sm, _ := json.MarshalIndent(sourceMarkets, "", "  ")
	om, _ := json.MarshalIndent(otherMarkets, "", "  ")
	user = fmt.Sprintf(`Source platform: %s
Other platform: %s
Inferred event title hint: %s

Source markets JSON:
%s

Other platform candidate markets JSON:
%s

If other markets array is empty, still return a valid JSON assessment explaining no cross match.`,
		sourceLabel, otherLabel, eventTitle, string(sm), string(om))
	return system, user
}
