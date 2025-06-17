package store

import (
	"fmt"
	"sync"

	"github.com/abgdnv/gocommerce/internal/product/errors"
)

// inMemory implements ProductStore using an in-memory map.
type inMemory struct {
	mu       sync.RWMutex
	products map[string]Product
	nextID   int
}

// NewInMemoryStore creates a new instance of ProductStore
func NewInMemoryStore() ProductStore {
	return &inMemory{
		products: make(map[string]Product),
		nextID:   1,
	}
}

// Product represents a product entity in the store.
type Product struct {
	ID    string
	Name  string
	Price int64 // Price in cents
	Stock int32
	/*  TODO:
	Description string
	sku         string // SKU (Stock Keeping Unit)
	imageURL    []string // Image URLs
	categories  []string
	active      bool
	*/
}

// FindByID retrieves a product by its ID.
func (s *inMemory) FindByID(id string) (*Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.products[id]
	if !ok {
		return nil, errors.ErrProductNotFound
	}
	return &t, nil
}

// FindAll retrieves all products.
func (s *inMemory) FindAll() (*[]Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Product, 0, len(s.products))
	for _, t := range s.products {
		list = append(list, t)
	}
	return &list, nil
}

// Create creates a new product and returns it.
func (s *inMemory) Create(name string, price int64, stock int32) (*Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	product := Product{
		ID:    fmt.Sprintf("%d", s.nextID),
		Name:  name,
		Price: price,
		Stock: stock,
	}
	s.nextID++
	s.products[product.ID] = product

	return &product, nil
}

// DeleteByID deletes a product by its ID.
func (s *inMemory) DeleteByID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.products[id]; !exists {
		return errors.ErrProductNotFound
	}
	delete(s.products, id)
	return nil
}
