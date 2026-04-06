package risk

import (
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/ports/input"
)

// MMEvaluator applies global HFT risk flags to MM quotes.
type MMEvaluator struct {
	root *config.Root
}

var _ input.RiskEvaluator = (*MMEvaluator)(nil)

func NewMMEvaluator(root *config.Root) *MMEvaluator {
	return &MMEvaluator{root: root}
}

func (e *MMEvaluator) IsSafe(quote domain.MMQuote, pos *domain.MMPosition, tox domain.ToxicitySignal) bool {
	_ = pos
	if e.root == nil {
		return false
	}
	if e.root.Hft.Risk.KillSwitch {
		return false
	}
	if tox.Unsafe {
		return false
	}
	if !quote.Bid.IsPositive() || !quote.Ask.IsPositive() {
		return false
	}
	if quote.Ask.LessThanOrEqual(quote.Bid) {
		return false
	}
	return true
}
