package input

import (
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/shopspring/decimal"
)

// QuotingEngine produces a two-sided quote from a snapshot, position, and toxicity.
type QuotingEngine interface {
	GenerateQuote(snapshot *domain.MarketSnapshot, pos *domain.MMPosition, tox domain.ToxicitySignal, tickSize decimal.Decimal) (domain.MMQuote, error)
}
