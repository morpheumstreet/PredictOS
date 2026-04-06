package marketmaker

import (
	"context"
	"errors"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/domain"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/ports/input"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/depth"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/toxicity"
	"github.com/shopspring/decimal"
)

// UseCase orchestrates snapshot, toxicity, quoting, and risk for one leg.
type UseCase struct {
	mdp   input.MarketDataProvider
	tox   input.ToxicityDetector
	qe    input.QuotingEngine
	risk  input.RiskEvaluator
	mmCfg config.MarketMakerCfg
}

func NewUseCase(
	mdp input.MarketDataProvider,
	tox input.ToxicityDetector,
	qe input.QuotingEngine,
	risk input.RiskEvaluator,
	mmCfg config.MarketMakerCfg,
) *UseCase {
	return &UseCase{mdp: mdp, tox: tox, qe: qe, risk: risk, mmCfg: mmCfg}
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
	if u.mmCfg.DepthPauseEnabled {
		pb, pa := depth.Pauses(&snap.Book, u.mmCfg)
		tox.PauseBidQuotes = pb
		tox.PauseAskQuotes = pa
	}
	if tox.PauseBidQuotes {
		return zero, false, tox, nil
	}
	if u.mmCfg.VpinEnabled {
		_, bad := toxicity.VPINUnsafe(snap.Trades, u.mmCfg.VpinMinTrades, u.mmCfg.VpinImbalanceThreshold)
		if bad {
			tox.Unsafe = true
		}
	}
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

// SubscribeL2 delegates to the market data provider (CLOB book updates → snapshots).
func (u *UseCase) SubscribeL2(ctx context.Context) (<-chan domain.MarketSnapshot, error) {
	if u == nil || u.mdp == nil {
		return nil, errors.New("marketmaker: nil use case or market data provider")
	}
	return u.mdp.SubscribeL2(ctx)
}
