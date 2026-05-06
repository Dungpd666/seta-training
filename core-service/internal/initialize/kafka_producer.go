package initialize

import (
	"context"
	"encoding/json"
	"sync"

	kafka "github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	brokers []string
	mu      sync.RWMutex
	writers map[string]*kafka.Writer
}

func NewKafkaProducer(brokers []string) *KafkaProducer {
	return &KafkaProducer{
		brokers: brokers,
		writers: make(map[string]*kafka.Writer),
	}
}

func (p *KafkaProducer) Publish(ctx context.Context, topic string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	p.mu.RLock()
	w, ok := p.writers[topic]
	p.mu.RUnlock()

	if !ok {
		p.mu.Lock()
		if w, ok = p.writers[topic]; !ok {
			w = kafka.NewWriter(kafka.WriterConfig{
				Brokers:  p.brokers,
				Topic:    topic,
				Balancer: &kafka.LeastBytes{},
			})
			p.writers[topic] = w
		}
		p.mu.Unlock()
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return w.WriteMessages(ctx, kafka.Message{Value: data})
}

func (p *KafkaProducer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, w := range p.writers {
		w.Close()
	}
}
