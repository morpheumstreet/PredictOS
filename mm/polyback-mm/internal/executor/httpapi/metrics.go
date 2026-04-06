package httpapi

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// OrderMetrics isolates Prometheus registration (SRP, testability).
type OrderMetrics struct {
	Placed prometheus.Counter
}

func NewOrderMetrics() *OrderMetrics {
	return &OrderMetrics{
		Placed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "polyback_executor_orders_placed_total",
			Help: "Limit/market orders accepted by executor API",
		}),
	}
}
