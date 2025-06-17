// Package service provides the implementation of product-related business logic.
package service

import (
	"fmt"
	"log"

	"github.com/abgdnv/gocommerce/internal/product/store"
)

// ProductService defines the methods for managing products.
type ProductService interface {
	GetProducts() ([]ProductDTO, error)
	GetProductByID(id string) (*ProductDTO, error)
	CreateProduct(product ProductDTO) (*ProductDTO, error)
	DeleteProductByID(id string) error
}

// service implements ProductService and provides methods to manage products.
type service struct {
	repository store.Store
}

// NewService creates a new instance of ProductService with the provided repository.
func NewService(repo store.Store) ProductService {
	return &service{
		repository: repo,
	}
}

// ProductDTO represents the data transfer object for a product.
type ProductDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name" validate:"required,max=100"`
	Price int64  `json:"price" validate:"required,min=0"`
	Stock int32  `json:"stock" validate:"required,min=0"`
}

// GetProductByID retrieves a product by its ID and returns it as a ProductDTO.
func (s *service) GetProductByID(id string) (*ProductDTO, error) {
	product, err := s.repository.GetProductByID(id)
	if err != nil {
		log.Printf("Error fetching product by ID %s: %v", id, err)
		return nil, fmt.Errorf("failed to fetch product by ID %s: %w", id, err)
	}

	return &ProductDTO{
		ID:    product.ID,
		Name:  product.Name,
		Price: product.Price,
		Stock: product.Stock,
	}, nil
}

// GetProducts retrieves a list of all products and returns them as ProductDTOs.
func (s *service) GetProducts() ([]ProductDTO, error) {
	products, err := s.repository.GetProducts()
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	productDTOs := make([]ProductDTO, len(products))

	for i, item := range products {
		productDTOs[i] = ProductDTO{
			ID:    item.ID,
			Name:  item.Name,
			Price: item.Price,
			Stock: item.Stock,
		}
	}

	return productDTOs, nil
}

// CreateProduct creates a new product and returns it as a ProductDTO.
func (s *service) CreateProduct(product ProductDTO) (*ProductDTO, error) {
	p, err := s.repository.CreateProduct(product.Name, product.Price, product.Stock)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return &ProductDTO{
		ID:    p.ID,
		Name:  p.Name,
		Price: p.Price,
		Stock: p.Stock,
	}, nil
}

// DeleteProductByID deletes a product by its ID.
func (s *service) DeleteProductByID(id string) error {
	return s.repository.DeleteProductByID(id)
}
