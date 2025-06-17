package repository

import (
	"fmt"
	"sync"

	"github.com/abgdnv/gocommerce/internal/product/errorsProduct"
)

// InMemoryProductRepository implements ProductRepositoryContract using an in-memory map.
type InMemoryProductRepository struct {
	mu       sync.RWMutex
	products map[string]Product
	nextID   int
}

func NewInMemoryProductRepository() *InMemoryProductRepository {
	return &InMemoryProductRepository{
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

func (s *InMemoryProductRepository) GetProductByID(id string) (*Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.products[id]
	if !ok {
		return nil, errorsProduct.ErrProductNotFound
	}
	return &t, nil
}

func (s *InMemoryProductRepository) DeleteProductByID(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.products[id]; !exists {
		return errorsProduct.ErrProductNotFound
	}
	delete(s.products, id)
	return nil
}

func (s *InMemoryProductRepository) GetProducts() ([]Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]Product, 0, len(s.products))
	for _, t := range s.products {
		list = append(list, t)
	}
	return list, nil
}

func (s *InMemoryProductRepository) CreateProduct(name string, price int64, stock int32) (Product, error) {
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
