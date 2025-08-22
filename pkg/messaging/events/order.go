package events

import (
	"encoding/json"
	"time"

	"github.com/abgdnv/gocommerce/pkg/messaging"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/propagation"
)

type OrderCreatedEvent struct {
	Carrier    propagation.MapCarrier `json:"carrier"`
	OrderID    uuid.UUID              `json:"order_id"`
	UserID     uuid.UUID              `json:"user_id"`
	TotalPrice int64                  `json:"total_price"`
	CreatedAt  time.Time              `json:"created_at"`
}

func (o OrderCreatedEvent) Subject() string {
	return messaging.OrdersCreatedSubject
}

func (o OrderCreatedEvent) Payload() ([]byte, error) {
	return json.Marshal(o)
}
