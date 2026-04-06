package ws

import (
	"context"
)

// EventListener is a placeholder for future news or calendar feeds that trigger risk-off (e.g. cancel-all).
// Wire OnAlert to strategy shutdown when an external feed is configured.
type EventListener struct {
	OnAlert func()
}

// Start blocks until ctx is cancelled. Override with a real poll / stream implementation later.
func (e *EventListener) Start(ctx context.Context) {
	<-ctx.Done()
}
