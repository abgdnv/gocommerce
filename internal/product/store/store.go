// Package store provides an interface for product storage operations.
package store

// ProductStore is an interface for product storage operations.
// It abstracts the underlying data store, allowing for different implementations (e.g., in-memory, database).
type ProductStore interface {
	// FindByID retrieves a single product by its unique identifier.
	// Returns ErrProductNotFound if no product exists with the given ID.
	FindByID(id string) (*Product, error)

	// FindAll returns all available products.
	// Returns an empty slice if no products exist.
	FindAll() (*[]Product, error)

	// Create adds a new product to the system.
	// Returns error if the product cannot be created.
	Create(name string, price int64, stock int32) (*Product, error)

	// DeleteByID removes a product by its ID.
	// Returns ErrProductNotFound if no product exists with the given ID.
	DeleteByID(id string) error
}
