// Package wiring holds composition-root adapters (DIP) between bounded contexts.
package wiring

import (
	"time"

	"github.com/profitlock/PredictOS/mm/polyback-mm/internal/hftevents"
	polyws "github.com/profitlock/PredictOS/mm/polyback-mm/internal/platforms/polymarket/ws"
)

type tobFromPublisher struct {
	p hftevents.Publisher
}

func (t tobFromPublisher) Enabled() bool {
	return t.p != nil && t.p.Enabled()
}

func (t tobFromPublisher) PublishAt(ts time.Time, eventType, key string, data any) {
	if t.p == nil {
		return
	}
	t.p.PublishAt(ts, eventType, key, data)
}

// TOBFromPublisher narrows the Kafka publisher to what the CLOB WebSocket client needs.
func TOBFromPublisher(p hftevents.Publisher) polyws.TOBEventEmitter {
	if p == nil {
		return nil
	}
	return tobFromPublisher{p: p}
}

var _ polyws.TOBEventEmitter = tobFromPublisher{}
