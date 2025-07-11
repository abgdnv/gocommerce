// Package store provides an interface for order storage operations.
package store

import (
	"context"

	"github.com/abgdnv/gocommerce/order_service/internal/store/db"
	"github.com/google/uuid"
)

// OrderStore is an interface for order storage operations.
// It abstracts the underlying data store, allowing for different implementations (e.g., in-memory, database).
type OrderStore interface {
	// FindByID retrieves a single order by its unique identifier.
	// Returns ErrOrderNotFound if no order exists with the given ID.
	FindByID(ctx context.Context, id uuid.UUID) (*db.Order, *[]db.OrderItem, error)

	// FindOrdersByUserID returns all available orders for a specific user.
	// Returns an empty slice if no orders exist.
	FindOrdersByUserID(ctx context.Context, params *db.FindOrdersByUserIDParams) (*[]db.Order, error)

	// CreateOrder adds a new order to the system.
	// Returns error if the order cannot be created.
	CreateOrder(ctx context.Context, orderParams *db.CreateOrderParams, items *[]db.CreateOrderItemParams) (*db.Order, *[]db.OrderItem, error)

	// Update modifies an existing order's details.
	// Returns ErrOrderNotFound if no order exists with the given ID and version.
	Update(ctx context.Context, params *db.UpdateOrderParams) (*db.Order, error)
}
