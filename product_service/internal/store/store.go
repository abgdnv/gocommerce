// Package store provides an interface for product storage operations.
package store

import (
	"context"

	"github.com/abgdnv/gocommerce/product_service/internal/store/db"
	"github.com/google/uuid"
)

// ProductStore is an interface for product storage operations.
// It abstracts the underlying data store, allowing for different implementations (e.g., in-memory, database).
type ProductStore interface {
	// FindByID retrieves a single product by its unique identifier.
	// Returns ErrProductNotFound if no product exists with the given ID.
	FindByID(ctx context.Context, id uuid.UUID) (*db.Product, error)

	// FindAll returns all available products.
	// Returns an empty slice if no products exist.
	FindAll(ctx context.Context, offset, limit int32) (*[]db.Product, error)

	// Create adds a new product to the system.
	// Returns error if the product cannot be created.
	Create(ctx context.Context, name string, price int64, stock int32) (*db.Product, error)

	// Update modifies an existing product's details.
	// Returns ErrProductNotFound if no product exists with the given ID and version.
	Update(ctx context.Context, id uuid.UUID, name string, price int64, stock int32, version int32) (*db.Product, error)

	// UpdateStock adjusts the stock quantity of a product.
	// Returns ErrProductNotFound if no product exists with the given ID and version.
	UpdateStock(ctx context.Context, id uuid.UUID, stock int32, version int32) (*db.Product, error)

	// DeleteByID removes a product by its ID.
	// Returns ErrProductNotFound if no product exists with the given ID.
	DeleteByID(ctx context.Context, id uuid.UUID, version int32) error
}
