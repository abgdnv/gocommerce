package store

import (
	"fmt"
	"sync"

	"github.com/abgdnv/gocommerce/internal/product/errors"
)

// inMemory implements Store using an in-memory map.
type inMemory struct {
	mu       sync.RWMutex
	products map[string]Product
	nextID   int
}

// NewInMemoryStore creates a new instance of Store
func NewInMemoryStore() Store {
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
	/*
		Description string
		sku         string // SKU (Stock Keeping Unit)
		imageURL    []string // Image URLs
		categories  []string
		active      bool
	*/
}

// GetProductByID retrieves a product by its ID.
func (s *inMemory) GetProductByID(id string) (*Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.products[id]
	if !ok {
		return nil, errors.ErrProductNotFound
	}
	return &t, nil
}

// GetProducts retrieves all products.
func (s *inMemory) GetProducts() ([]Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Product, 0, len(s.products))
	for _, t := range s.products {
		list = append(list, t)
	}
	return list, nil
}

// CreateProduct creates a new product and returns it.
func (s *inMemory) CreateProduct(name string, price int64, stock int32) (Product, error) {
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

	return product, nil
}

// DeleteProductByID deletes a product by its ID.
func (s *inMemory) DeleteProductByID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.products[id]; !exists {
		return errors.ErrProductNotFound
	}
	delete(s.products, id)
	return nil
}
