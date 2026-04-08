package app

import (
	"net/http"
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/config"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/adapters/polyfactual"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/adapters/x402svc"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/llm"
	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/intelligence/usecase"
)

// Deps wires intelligence HTTP handlers (constructed in cmd/intelligence).
type Deps struct {
	Root *config.Root
	HTTP *http.Client

	Polyfactual *polyfactual.Client
	X402        *x402svc.Service
	LLM         *llm.Facade

	GetEvents       *usecase.GetEvents
	EventAnalysis   *usecase.EventAnalysis
	AnalyzeMarkets  *usecase.AnalyzeEventMarkets
	Bookmaker       *usecase.Bookmaker
	ArbitrageFinder *usecase.ArbitrageFinder
	Mapper          *usecase.Mapper
	Trading         *usecase.Trading
}

// NewDeps builds defaults from config and environment.
func NewDeps(root *config.Root) *Deps {
	if root == nil {
		root = &config.Root{}
	}
	hc := &http.Client{Timeout: 120 * time.Second}
	pf := polyfactual.NewClient(hc)
	x4 := x402svc.NewService(hc)
	llmFacade := llm.NewFacade(hc)

	return &Deps{
		Root:            root,
		HTTP:            hc,
		Polyfactual:     pf,
		X402:            x4,
		LLM:             llmFacade,
		GetEvents:       usecase.NewGetEvents(root, hc),
		EventAnalysis:   usecase.NewEventAnalysis(llmFacade),
		AnalyzeMarkets:  usecase.NewAnalyzeEventMarkets(usecase.NewGetEvents(root, hc), llmFacade),
		Bookmaker:       usecase.NewBookmaker(llmFacade),
		ArbitrageFinder: usecase.NewArbitrageFinder(root, hc, llmFacade),
		Mapper:          usecase.NewMapper(),
		Trading:         usecase.NewTrading(root, hc),
	}
}
