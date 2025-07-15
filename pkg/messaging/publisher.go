package messaging

import (
	"context"
)

type Event interface {
	Subject() string
	Payload() ([]byte, error)
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}
