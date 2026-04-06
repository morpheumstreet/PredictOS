package input

import "github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"

// ToxicityDetector scores flow toxicity from recent trades and book state.
type ToxicityDetector interface {
	Assess(trades []domain.Trade, book *domain.OrderBookL2) domain.ToxicitySignal
}
