package hftevents

import (
	"encoding/json"
	"time"
)

type Publisher interface {
	Enabled() bool
	Publish(eventType string, key string, data any)
	PublishAt(ts time.Time, eventType string, key string, data any)
	Close() error
}

type envelope struct {
	TS     time.Time `json:"ts"`
	Source string    `json:"source"`
	Type   string    `json:"type"`
	Data   any       `json:"data"`
}

func MarshalEnvelope(source string, ts time.Time, typ string, data any) ([]byte, error) {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return json.Marshal(envelope{
		TS:     ts.UTC(),
		Source: source,
		Type:   typ,
		Data:   data,
	})
}

// NoopPublisher implements Publisher with no-op sends.
type NoopPublisher struct {
	On bool
}

func (n *NoopPublisher) Enabled() bool { return n.On }

func (n *NoopPublisher) Publish(eventType string, key string, data any) {
	n.PublishAt(time.Time{}, eventType, key, data)
}

func (n *NoopPublisher) PublishAt(ts time.Time, eventType string, key string, data any) {}

func (n *NoopPublisher) Close() error { return nil }
