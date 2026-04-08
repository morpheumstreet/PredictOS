package mapping

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

const (
	defaultTick = "0.01"
	minShares   = 5
	minBudget   = 1.0
	maxBudget   = 100.0
)

// MarketData matches mapper-agent market payload (subset).
type MarketData struct {
	Title            string `json:"title"`
	Question         string `json:"question"`
	Slug             string `json:"slug"`
	ConditionID      string `json:"conditionId"`
	ClobTokenIds     string `json:"clobTokenIds"`
	Outcomes         string `json:"outcomes"`
	OutcomePrices    string `json:"outcomePrices"`
	MinimumTickSize  string `json:"minimumTickSize"`
	NegRisk          *bool  `json:"negRisk"`
	Closed           bool   `json:"closed"`
	AcceptingOrders  *bool  `json:"acceptingOrders"`
}

// AnalysisResult subset for mapping.
type AnalysisResult struct {
	RecommendedAction string `json:"recommendedAction"`
}

// PolymarketOrderParams output shape.
type PolymarketOrderParams struct {
	TokenID          string  `json:"tokenId"`
	Price            float64 `json:"price"`
	Side             string  `json:"side"`
	Size             int     `json:"size"`
	FeeRateBps       int     `json:"feeRateBps"`
	TickSize         string  `json:"tickSize"`
	NegRisk          bool    `json:"negRisk"`
	ConditionID      string  `json:"conditionId"`
	MarketSlug       string  `json:"marketSlug"`
	OrderDescription string  `json:"orderDescription"`
}

func parseTokenPair(clob string) (string, string, error) {
	var arr []string
	if err := json.Unmarshal([]byte(clob), &arr); err != nil {
		return "", "", fmt.Errorf("clobTokenIds: %w", err)
	}
	if len(arr) < 2 {
		return "", "", fmt.Errorf("clobTokenIds need 2 tokens")
	}
	return arr[0], arr[1], nil
}

func parseOutcomes(s string) []string {
	if s == "" {
		s = `["Yes","No"]`
	}
	var o []string
	if json.Unmarshal([]byte(s), &o) != nil {
		return []string{"Yes", "No"}
	}
	return o
}

func parsePrices(s string) []float64 {
	if s == "" {
		s = `["0.5","0.5"]`
	}
	var raw []string
	if json.Unmarshal([]byte(s), &raw) != nil {
		return []float64{0.5, 0.5}
	}
	out := make([]float64, len(raw))
	for i, x := range raw {
		var f float64
		_, _ = fmt.Sscanf(x, "%f", &f)
		out[i] = f
	}
	return out
}

func roundToTick(price float64, tickSize string) float64 {
	var tick float64
	_, _ = fmt.Sscanf(strings.TrimSpace(tickSize), "%f", &tick)
	if tick <= 0 {
		tick = 0.01
	}
	return math.Round(price/tick) * tick
}

// MapPolymarketOrder maps analysis + market to order params (Polymarket only).
func MapPolymarketOrder(a AnalysisResult, m MarketData, budgetUsd float64) (PolymarketOrderParams, error) {
	if m.Closed {
		return PolymarketOrderParams{}, fmt.Errorf("Market is closed")
	}
	if m.AcceptingOrders != nil && !*m.AcceptingOrders {
		return PolymarketOrderParams{}, fmt.Errorf("Market is not accepting orders")
	}
	if strings.TrimSpace(m.ClobTokenIds) == "" {
		return PolymarketOrderParams{}, fmt.Errorf("Missing clobTokenIds in market data")
	}
	t0, t1, err := parseTokenPair(m.ClobTokenIds)
	if err != nil {
		return PolymarketOrderParams{}, err
	}
	outcomes := parseOutcomes(m.Outcomes)
	prices := parsePrices(m.OutcomePrices)

	buyYes := strings.TrimSpace(a.RecommendedAction) == "BUY YES"
	yesIdx := -1
	noIdx := -1
	for i, o := range outcomes {
		ol := strings.ToLower(o)
		if ol == "yes" || ol == "up" {
			yesIdx = i
		}
		if ol == "no" || ol == "down" {
			noIdx = i
		}
	}
	var tokenID string
	var cur float64
	if buyYes {
		if yesIdx == 0 {
			tokenID = t0
		} else {
			tokenID = t1
		}
		if yesIdx >= 0 && yesIdx < len(prices) {
			cur = prices[yesIdx]
		} else {
			cur = prices[0]
		}
	} else {
		if noIdx == 0 {
			tokenID = t0
		} else {
			tokenID = t1
		}
		if noIdx >= 0 && noIdx < len(prices) {
			cur = prices[noIdx]
		} else if len(prices) > 1 {
			cur = prices[1]
		}
	}

	tick := m.MinimumTickSize
	if strings.TrimSpace(tick) == "" {
		tick = defaultTick
	}
	neg := false
	if m.NegRisk != nil {
		neg = *m.NegRisk
	}
	orderPrice := roundToTick(cur, tick)
	rawSize := budgetUsd / orderPrice
	size := int(math.Floor(rawSize))
	if size < minShares {
		return PolymarketOrderParams{}, fmt.Errorf("Budget too small. At current price (%.1f%%), minimum budget is $%.2f for %d shares", orderPrice*100, minShares*orderPrice, minShares)
	}
	title := m.Title
	if title == "" {
		title = m.Question
	}
	if title == "" {
		title = "Unknown Market"
	}
	desc := fmt.Sprintf("BUY %d shares @ %.1f%% for ~$%.2f on %q", size, orderPrice*100, float64(size)*orderPrice, title)
	return PolymarketOrderParams{
		TokenID:          tokenID,
		Price:            orderPrice,
		Side:             "BUY",
		Size:             size,
		FeeRateBps:       0,
		TickSize:         tick,
		NegRisk:          neg,
		ConditionID:      m.ConditionID,
		MarketSlug:       m.Slug,
		OrderDescription: desc,
	}, nil
}

// ValidateBudget checks mapper budget bounds.
func ValidateBudget(budget float64) error {
	if budget < minBudget || budget > maxBudget {
		return fmt.Errorf("Invalid budgetUsd. Must be between $%.0f and $%.0f", minBudget, maxBudget)
	}
	return nil
}
