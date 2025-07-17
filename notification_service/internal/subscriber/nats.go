package subscriber

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/messaging/events"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sync/errgroup"
)

// Start initializes the NATS JetStream consumer and starts multiple worker goroutines to process messages.
func Start(ctx context.Context, js jetstream.JetStream, subscriberCfg config.SubscriberConfig, logger *slog.Logger) error {
	cfg := jetstream.ConsumerConfig{
		FilterSubject: subscriberCfg.Subject,
		Durable:       subscriberCfg.Consumer,
		AckPolicy:     jetstream.AckExplicitPolicy,
	}
	consumer, err := js.CreateOrUpdateConsumer(ctx, subscriberCfg.Stream, cfg)
	if err != nil {
		return err
	}
	g, gCtx := errgroup.WithContext(ctx)
	for i := 0; i < subscriberCfg.Workers; i++ {
		g.Go(func() error {
			return runWorker(gCtx, consumer, subscriberCfg.Timeout, subscriberCfg.Interval, logger)
		})
	}
	return g.Wait()
}

// runWorker fetches messages from the NATS JetStream consumer and processes them.
func runWorker(ctx context.Context, consumer jetstream.Consumer, timeout time.Duration, interval time.Duration, logger *slog.Logger) error {
	for {
		select {
		case <-ctx.Done():
			// ctx was cancelled or timed out (e.g., application shutdown)
			return ctx.Err()
		default:
			batch, err := consumer.Fetch(1, jetstream.FetchMaxWait(timeout))
			if err != nil {
				// if the error is a timeout, we can just continue to the next iteration
				if errors.Is(err, nats.ErrTimeout) {
					continue
				}
				slog.Error("failed to fetch messages", "error", err)
				// for other errors, we can log and retry
				time.Sleep(interval)
				continue
			}
			for msg := range batch.Messages() {
				handleMessage(msg, logger)
			}
		}
	}
}

// handleMessage processes a single message from the NATS JetStream consumer.
func handleMessage(msg jetstream.Msg, logger *slog.Logger) {
	if msg == nil {
		slog.Error("received nil message")
		return
	}
	var event events.OrderCreatedEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		slog.Error("failed to unmarshal message", "error", err, "subject", msg.Subject())
		if err := msg.Nak(); err != nil {
			slog.Error("failed to nack message", "error", err)
		}
		return
	}

	slog.Info("received order created event",
		slog.String("subject", msg.Subject()),
		slog.String("order_id", event.OrderID.String()),
		slog.String("user_id", event.UserID.String()),
		slog.String("created_at", event.CreatedAt.Format(time.RFC3339)))

	notificationJob()

	if err := msg.Ack(); err != nil {
		slog.Error("failed to ack message", "error", err)
	}
}

// notificationJob simulates a job that processes the notification.
func notificationJob() {
	// simulate some processing time
	time.Sleep(100 * time.Millisecond)
}
