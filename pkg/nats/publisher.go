package nats

import (
	"context"
	"fmt"

	"github.com/abgdnv/gocommerce/pkg/messaging"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsPublisher struct {
	js jetstream.JetStream
}

func NewNatsPublisher(js jetstream.JetStream) *NatsPublisher {
	return &NatsPublisher{js: js}
}

func (p *NatsPublisher) Publish(ctx context.Context, event messaging.Event) error {
	data, err := event.Payload()
	if err != nil {
		return fmt.Errorf("failed to get event payload: %w", err)
	}
	_, err = p.js.Publish(ctx, event.Subject(), data)
	return err
}
