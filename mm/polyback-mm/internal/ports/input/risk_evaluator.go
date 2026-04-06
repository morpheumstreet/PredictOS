package input

import "github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"

// RiskEvaluator gates quoting from global risk config and toxicity veto.
type RiskEvaluator interface {
	IsSafe(quote domain.MMQuote, pos *domain.MMPosition, tox domain.ToxicitySignal) bool
}
