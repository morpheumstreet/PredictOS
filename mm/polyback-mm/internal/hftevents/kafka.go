package hftevents

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaPublisher struct {
	on       bool
	topic    string
	source   string
	client   *kgo.Client
	mu       sync.Mutex
	closed   bool
}

func NewKafkaPublisher(brokers []string, topic, source string, enabled bool) (*KafkaPublisher, error) {
	if !enabled || len(brokers) == 0 {
		return &KafkaPublisher{on: false}, nil
	}
	t := strings.TrimSpace(topic)
	if t == "" {
		t = "polybot.events"
	}
	src := strings.TrimSpace(source)
	if src == "" {
		src = "app"
	}
	cl, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.AllowAutoTopicCreation(),
	)
	if err != nil {
		return nil, err
	}
	return &KafkaPublisher{
		on:     true,
		topic:  t,
		source: strings.ToLower(src),
		client: cl,
	}, nil
}

func (k *KafkaPublisher) Enabled() bool { return k != nil && k.on && k.client != nil }

func (k *KafkaPublisher) Publish(eventType string, key string, data any) {
	k.PublishAt(time.Time{}, eventType, key, data)
}

func (k *KafkaPublisher) PublishAt(ts time.Time, eventType string, key string, data any) {
	if !k.Enabled() {
		return
	}
	if eventType == "" {
		return
	}
	payload, err := MarshalEnvelope(k.source, ts, strings.TrimSpace(eventType), data)
	if err != nil {
		return
	}
	rec := kgo.Record{Topic: k.topic, Value: payload}
	if key != "" {
		rec.Key = []byte(key)
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.closed || k.client == nil {
		return
	}
	k.client.Produce(context.Background(), &rec, nil)
}

func (k *KafkaPublisher) Close() error {
	if k == nil || k.client == nil {
		return nil
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.closed {
		return nil
	}
	k.closed = true
	k.client.Close()
	return nil
}

func NewPublisherFromBrokers(brokers []string, topic, source string, enabled bool) (Publisher, error) {
	kp, err := NewKafkaPublisher(brokers, topic, source, enabled)
	if err != nil {
		return nil, err
	}
	if kp.Enabled() {
		return kp, nil
	}
	return &NoopPublisher{On: false}, nil
}
