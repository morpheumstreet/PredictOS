package arbitragefee

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// Config mirrors TS getArbitrageFeeConfig.
type Config struct {
	PolymarketFeeBps int
	KalshiFeeBps     int
	MinNetProfitUsd  float64
}

func LoadConfig() Config {
	return Config{
		PolymarketFeeBps: readInt("ARBITRAGE_POLYMARKET_FEE_BPS", 0),
		KalshiFeeBps:     readInt("ARBITRAGE_KALSHI_FEE_BPS", 0),
		MinNetProfitUsd:  readFloat("ARBITRAGE_MIN_NET_PROFIT_USD", 0),
	}
}

func readInt(name string, def int) int {
	s := strings.TrimSpace(os.Getenv(name))
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func readFloat(name string, def float64) float64 {
	s := strings.TrimSpace(os.Getenv(name))
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return def
	}
	return f
}

// Enrich mutates strategy map with feeAdjusted block (gross fields unchanged).
func Enrich(arbitrage map[string]any, cfg Config) {
	strategy, ok := arbitrage["strategy"].(map[string]any)
	if !ok || strategy == nil {
		return
	}
	buyYesOn, _ := strategy["buyYesOn"].(string)
	buyNoOn, _ := strategy["buyNoOn"].(string)
	buyYesPrice := num(strategy["buyYesPrice"])
	buyNoPrice := num(strategy["buyNoPrice"])
	totalCost := num(strategy["totalCost"])
	guaranteed := num(strategy["guaranteedPayout"])

	yesBps := cfg.KalshiFeeBps
	if buyYesOn == "polymarket" {
		yesBps = cfg.PolymarketFeeBps
	}
	noBps := cfg.KalshiFeeBps
	if buyNoOn == "polymarket" {
		noBps = cfg.PolymarketFeeBps
	}
	feeYes := buyYesPrice * (float64(yesBps) / 10000)
	feeNo := buyNoPrice * (float64(noBps) / 10000)
	totalFees := feeYes + feeNo
	totalAfter := totalCost + totalFees
	net := guaranteed - totalAfter
	var pct *float64
	if totalAfter > 0 {
		p := (net / totalAfter) * 100
		pct = &p
	}
	arbitrage["feeAdjusted"] = map[string]any{
		"polymarketFeeBps":      cfg.PolymarketFeeBps,
		"kalshiFeeBps":          cfg.KalshiFeeBps,
		"minNetProfitUsd":       cfg.MinNetProfitUsd,
		"estimatedFeeYes":     feeYes,
		"estimatedFeeNo":      feeNo,
		"totalFees":           totalFees,
		"totalCostAfterFees":  totalAfter,
		"netProfitAfterFees":  net,
		"profitPercentAfterFees": pct,
		"viableAfterFees":     net > cfg.MinNetProfitUsd,
	}
}

func num(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}
