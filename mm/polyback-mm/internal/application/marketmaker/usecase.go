package marketmaker

import (
	"context"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/ports/input"
	"github.com/shopspring/decimal"
)

// UseCase orchestrates snapshot, toxicity, quoting, and risk for one leg.
type UseCase struct {
	mdp  input.MarketDataProvider
	tox  input.ToxicityDetector
	qe   input.QuotingEngine
	risk input.RiskEvaluator
}

func NewUseCase(
	mdp input.MarketDataProvider,
	tox input.ToxicityDetector,
	qe input.QuotingEngine,
	risk input.RiskEvaluator,
) *UseCase {
	return &UseCase{mdp: mdp, tox: tox, qe: qe, risk: risk}
}

// MakerBid returns a risk-checked bid for maker buy logic. If the MM path is unavailable or unsafe, ok is false (caller may fall back to legacy pricing).
func (u *UseCase) MakerBid(ctx context.Context, tokenID string, tickSize decimal.Decimal, pos *domain.MMPosition) (bid decimal.Decimal, ok bool, tox domain.ToxicitySignal, err error) {
	var zero decimal.Decimal
	if u == nil || u.mdp == nil || u.qe == nil || u.tox == nil || u.risk == nil {
		return zero, false, domain.ToxicitySignal{}, nil
	}
	snap, have := u.mdp.Snapshot(ctx, tokenID)
	if !have {
		return zero, false, domain.ToxicitySignal{}, nil
	}
	tox = u.tox.Assess(snap.Trades, &snap.Book)
	if tox.Unsafe {
		return zero, false, tox, nil
	}
	q, err := u.qe.GenerateQuote(&snap, pos, tox, tickSize)
	if err != nil {
		return zero, false, tox, err
	}
	if !u.risk.IsSafe(q, pos, tox) {
		return zero, false, tox, nil
	}
	return q.Bid, true, tox, nil
}
