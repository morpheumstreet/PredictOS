package ws

import "time"

// MarketFeed is the read/subscribe surface strategies and simulators need (ISP).
type MarketFeed interface {
	GetTopOfBook(assetID string) (*TopOfBook, bool)
	SubscribeAssets(ids []string)
}

// TOBEventEmitter publishes top-of-book snapshots to external buses (Kafka).
// Separated from the full hftevents.Publisher so the WS client does not depend on Close/Publish.
type TOBEventEmitter interface {
	Enabled() bool
	PublishAt(ts time.Time, eventType string, key string, data any)
}

// FeedController starts and stops the WebSocket runtime (composition root wires this).
type FeedController interface {
	MarketFeed
	StartBackground()
	Close()
}

var (
	_ MarketFeed      = (*ClobClient)(nil)
	_ FeedController  = (*ClobClient)(nil)
	_ TOBEventEmitter = noopTOB{}
)

// noopTOB is the default TOB sink when none is injected (no Kafka coupling).
type noopTOB struct{}

func (noopTOB) Enabled() bool { return false }

func (noopTOB) PublishAt(time.Time, string, string, any) {}

// NoopMarketFeed satisfies MarketFeed when no WS is wired (safe defaults).
type NoopMarketFeed struct{}

func (NoopMarketFeed) GetTopOfBook(string) (*TopOfBook, bool) { return nil, false }

func (NoopMarketFeed) SubscribeAssets([]string) {}

var _ MarketFeed = NoopMarketFeed{}
