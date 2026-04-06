package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shopspring/decimal"
)

type Service struct {
	bankroll prometheus.Gauge
	exposure prometheus.Gauge
}

func New() *Service {
	return &Service{
		bankroll: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "polyback_strategy_bankroll_usd",
			Help: "Effective bankroll used for sizing",
		}),
		exposure: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "polyback_strategy_total_exposure_usd",
			Help: "Estimated open + unhedged exposure",
		}),
	}
}

func (s *Service) UpdateBankroll(d decimal.Decimal) {
	f, _ := d.Float64()
	s.bankroll.Set(f)
}

func (s *Service) UpdateTotalExposure(d decimal.Decimal) {
	f, _ := d.Float64()
	s.exposure.Set(f)
}
