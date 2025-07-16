// Package service provides the implementation of product-related business logic.
package service

import (
	"context"
	"fmt"

	"github.com/abgdnv/gocommerce/product_service/internal/store"
	"github.com/abgdnv/gocommerce/product_service/internal/store/db"
	"github.com/google/uuid"
)

// ProductService defines the methods for managing products.
// It abstracts the underlying business logic and data access.
type ProductService interface {
	// FindByID retrieves a single product by its unique identifier.
	// Returns ErrProductNotFound if no product exists with the given ID.
	FindByID(ctx context.Context, id uuid.UUID) (*ProductDto, error)

	// FindByIDs returns products by IDs.
	// Returns an empty slice if no products exist.
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]ProductDto, error)

	// FindAll returns all available products.
	// Returns an empty slice if no products exist.
	FindAll(ctx context.Context, offset, limit int32) ([]ProductDto, error)

	// Create adds a new product to the system.
	// Returns error if the product cannot be created.
	Create(ctx context.Context, product ProductCreateDto) (*ProductDto, error)

	// Update modifies an existing product's details.
	// Returns ErrProductNotFound if no product exists with the given ID and version.
	Update(ctx context.Context, product ProductDto) (*ProductDto, error)

	// UpdateStock adjusts the stock quantity of a product.
	// Returns ErrProductNotFound if no product exists with the given ID and version.
	UpdateStock(ctx context.Context, id uuid.UUID, stock int32, version int32) (*ProductDto, error)

	// DeleteByID removes a product by its ID.
	// Returns ErrProductNotFound if no product exists with the given ID.
	DeleteByID(ctx context.Context, id uuid.UUID, version int32) error
}

// Service implements ProductService and provides methods to manage products.
type Service struct {
	repository store.ProductStore
}

// NewService creates a new instance of ProductService with the provided repository.
func NewService(repo store.ProductStore) *Service {
	return &Service{
		repository: repo,
	}
}

// ProductCreateDto represents the data transfer object for creating a new product.
type ProductCreateDto struct {
	Name  string `json:"name"    validate:"required,max=100"`
	Price int64  `json:"price"   validate:"required,min=0"`
	Stock int32  `json:"stock"   validate:"required,min=0"`
}

// ProductDto represents the data transfer object for a product.
// Version is read-only and used for optimistic concurrency control.
type ProductDto struct {
	ID      string `json:"id"`
	Name    string `json:"name"    validate:"required,max=100"`
	Price   int64  `json:"price"   validate:"required,min=0"`
	Stock   int32  `json:"stock"   validate:"required,min=0"`
	Version int32  `json:"version" validate:"required,min=1"`
}

// StockUpdateDto represents the data transfer object for updating product stock.
type StockUpdateDto struct {
	Stock   int32 `json:"stock"   validate:"required,min=0"`
	Version int32 `json:"version" validate:"required,min=1"`
}

// FindByID retrieves a product by its ID and returns it as a ProductDto.
// Returns ErrProductNotFound if no product exists with the given ID.
func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*ProductDto, error) {
	product, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch product by ID %s: %w", id, err)
	}

	return toDto(product), nil
}

// FindByIDs retrieves a list of products and returns them as ProductDTOs.
// Returns an empty slice if no products exist or error if the retrieval fails.
func (s *Service) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]ProductDto, error) {
	products, err := s.repository.FindByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	productDTOs := make([]ProductDto, len(products))

	for i, item := range products {
		productDTOs[i] = *toDto(&item)
	}

	return productDTOs, nil
}

// FindAll retrieves a list of all products and returns them as ProductDTOs.
// Returns an empty slice if no products exist or error if the retrieval fails.
func (s *Service) FindAll(ctx context.Context, offset, limit int32) ([]ProductDto, error) {
	products, err := s.repository.FindAll(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	productDTOs := make([]ProductDto, len(products))

	for i, item := range products {
		productDTOs[i] = *toDto(&item)
	}

	return productDTOs, nil
}

// Create creates a new product and returns it as a ProductDto.
// Returns an error if the product cannot be created.
func (s *Service) Create(ctx context.Context, product ProductCreateDto) (*ProductDto, error) {
	p, err := s.repository.Create(ctx, product.Name, product.Price, product.Stock)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return toDto(p), nil
}

// Update modifies an existing product's details and returns the updated product as a ProductDto.
// Returns ErrProductNotFound if no product exists with the given ID and version.
func (s *Service) Update(ctx context.Context, product ProductDto) (*ProductDto, error) {
	updated, err := s.repository.Update(
		ctx,
		uuid.MustParse(product.ID),
		product.Name,
		product.Price,
		product.Stock,
		product.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to update product with ID %s: %w", product.ID, err)
	}

	return toDto(updated), nil
}

// UpdateStock adjusts the stock quantity of a product and returns the updated product as a ProductDto.
// Returns ErrProductNotFound if no product exists with the given ID and version.
func (s *Service) UpdateStock(ctx context.Context, id uuid.UUID, stock int32, version int32) (*ProductDto, error) {
	product, err := s.repository.UpdateStock(ctx, id, stock, version)
	if err != nil {
		return nil, fmt.Errorf("failed to update stock for product with ID %s: %w", id, err)
	}

	return toDto(product), nil
}

// DeleteByID deletes a product by its ID.
// Returns ErrProductNotFound if no product exists with the given ID and version.
func (s *Service) DeleteByID(ctx context.Context, id uuid.UUID, version int32) error {
	return s.repository.DeleteByID(ctx, id, version)
}

// toDto converts a store.Product to a ProductDto.
func toDto(product *db.Product) *ProductDto {
	return &ProductDto{
		ID:      product.ID.String(),
		Name:    product.Name,
		Price:   product.Price,
		Stock:   product.StockQuantity,
		Version: product.Version,
	}
}
