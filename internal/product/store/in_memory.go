package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/abgdnv/gocommerce/internal/product/errors"
	"github.com/abgdnv/gocommerce/internal/product/store/db"
	"github.com/google/uuid"
)

// inMemory implements ProductStore using an in-memory map.
type inMemory struct {
	mu       sync.RWMutex
	products map[uuid.UUID]db.Product
}

// NewInMemoryStore creates a new instance of ProductStore
func NewInMemoryStore() ProductStore {
	return &inMemory{
		products: make(map[uuid.UUID]db.Product),
	}
}

// FindByID retrieves a product by its ID.
func (s *inMemory) FindByID(_ context.Context, id uuid.UUID) (*db.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.products[id]
	if !ok {
		return nil, errors.ErrProductNotFound
	}
	return &t, nil
}

// FindAll retrieves all products.
func (s *inMemory) FindAll(_ context.Context, _ int32, _ int32) (*[]db.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]db.Product, 0, len(s.products))
	for _, t := range s.products {
		list = append(list, t)
	}
	return &list, nil
}

// Create creates a new product and returns it.
func (s *inMemory) Create(_ context.Context, name string, price int64, stock int32) (*db.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product := db.Product{
		ID:            uuid.New(),
		Name:          name,
		Price:         price,
		StockQuantity: stock,
	}
	s.products[product.ID] = product

	return &product, nil
}

// Update modifies an existing product and returns the updated product.
func (s *inMemory) Update(ctx context.Context, id uuid.UUID, name string, price int64, stock int32, version int32) (*db.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, exists := s.products[id]
	if !exists || stored.Version != version {
		return nil, errors.ErrProductNotFound
	}

	product := db.Product{
		ID:            id,
		Name:          name,
		Price:         price,
		StockQuantity: stock,
		Version:       version + 1,
	}
	s.products[id] = product

	return &product, nil
}

// UpdateStock updates the stock quantity of a product and returns the updated product.
func (s *inMemory) UpdateStock(ctx context.Context, id uuid.UUID, stock int32, version int32) (*db.Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, exists := s.products[id]
	if !exists || stored.Version != version {
		return nil, errors.ErrProductNotFound
	}

	product := db.Product{
		ID:            id,
		Name:          stored.Name,
		Price:         stored.Price,
		StockQuantity: stock,
		Version:       version + 1,
	}
	s.products[id] = product

	return &product, nil
}

// DeleteByID deletes a product by its ID.
func (s *inMemory) DeleteByID(_ context.Context, id uuid.UUID, version int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored, exists := s.products[id]
	if !exists {
		return errors.ErrProductNotFound
	}
	if stored.Version != version {
		return fmt.Errorf("version mismatch: expected %d, got %d", stored.Version, version)
	}

	delete(s.products, id)
	return nil
}
