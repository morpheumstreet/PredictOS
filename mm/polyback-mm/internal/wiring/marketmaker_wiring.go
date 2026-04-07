package wiring

import (
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/adapters/marketdata"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/application/marketmaker"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/ws"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/quoting"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/risk"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/toxicity"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/strategy/volatility"
)

// MarketMakerBundle wires study.md MM use case from the CLOB WebSocket client.
type MarketMakerBundle struct {
	UseCase *marketmaker.UseCase
	MDP     *marketdata.WSProvider
}

// NewMarketMakerBundle returns nil UseCase if clob is nil.
func NewMarketMakerBundle(root *config.Root, clob *polyws.ClobClient) *MarketMakerBundle {
	if clob == nil {
		return &MarketMakerBundle{UseCase: nil, MDP: nil}
	}
	mm := root.Hft.Strategy.MarketMaker
	mdp := marketdata.NewWSProvider(clob, 0)
	tox := toxicity.NewDetector(mm)
	volTr := volatility.NewTracker(mm)
	qe := quoting.NewEngine(mm, volTr)
	riskEv := risk.NewMMEvaluator(root)
	uc := marketmaker.NewUseCase(mdp, tox, qe, riskEv, mm)
	return &MarketMakerBundle{UseCase: uc, MDP: mdp}
}
