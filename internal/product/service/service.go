// Package service provides the implementation of product-related business logic.
package service

import (
	"fmt"
	"log"

	"github.com/abgdnv/gocommerce/internal/product/store"
)

// ProductService defines the methods for managing products.
// It abstracts the underlying business logic and data access.
type ProductService interface {
	// FindByID retrieves a single product by its unique identifier.
	// Returns ErrProductNotFound if no product exists with the given ID.
	FindByID(id string) (*ProductDto, error)

	// FindAll returns all available products.
	// Returns an empty slice if no products exist.
	FindAll() (*[]ProductDto, error)

	// Create adds a new product to the system.
	// Returns error if the product cannot be created.
	Create(product ProductDto) (*ProductDto, error)

	// DeleteByID removes a product by its ID.
	// Returns ErrProductNotFound if no product exists with the given ID.
	DeleteByID(id string) error
}

// service implements ProductService and provides methods to manage products.
type service struct {
	repository store.ProductStore
}

// NewService creates a new instance of ProductService with the provided repository.
func NewService(repo store.ProductStore) ProductService {
	return &service{
		repository: repo,
	}
}

// ProductDto represents the data transfer object for a product.
type ProductDto struct {
	ID    string `json:"id"`
	Name  string `json:"name" validate:"required,max=100"`
	Price int64  `json:"price" validate:"required,min=0"`
	Stock int32  `json:"stock" validate:"required,min=0"`
}

// FindByID retrieves a product by its ID and returns it as a ProductDto.
func (s *service) FindByID(id string) (*ProductDto, error) {
	product, err := s.repository.FindByID(id)
	if err != nil {
		log.Printf("Error fetching product by ID %s: %v", id, err)
		return nil, fmt.Errorf("failed to fetch product by ID %s: %w", id, err)
	}

	return toDto(product), nil
}

// FindAll retrieves a list of all products and returns them as ProductDTOs.
func (s *service) FindAll() (*[]ProductDto, error) {
	products, err := s.repository.FindAll()
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	productDTOs := make([]ProductDto, len(*products))

	for i, item := range *products {
		productDTOs[i] = *toDto(&item)
	}

	return &productDTOs, nil
}

// Create creates a new product and returns it as a ProductDto.
func (s *service) Create(product ProductDto) (*ProductDto, error) {
	p, err := s.repository.Create(product.Name, product.Price, product.Stock)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return toDto(p), nil
}

// DeleteByID deletes a product by its ID.
func (s *service) DeleteByID(id string) error {
	return s.repository.DeleteByID(id)
}

// toDto converts a store.Product to a ProductDto.
func toDto(product *store.Product) *ProductDto {
	return &ProductDto{
		ID:    product.ID,
		Name:  product.Name,
		Price: product.Price,
		Stock: product.Stock,
	}
}
