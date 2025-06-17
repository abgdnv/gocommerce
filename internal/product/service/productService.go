package service

import (
	"fmt"
	"log"

	"github.com/abgdnv/gocommerce/internal/product/repository"
)

// ProductService implements ProductServiceContract and provides methods to manage products.
type ProductService struct {
	repository repository.ProductRepositoryContract
}
type ProductDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name" validate:"required,max=100"`
	Price int64  `json:"price" validate:"required,min=0"`
	Stock int32  `json:"stock" validate:"required,min=0"`
}

func NewProductService(repo repository.ProductRepositoryContract) *ProductService {
	return &ProductService{
		repository: repo,
	}
}

func (s *ProductService) GetProducts() ([]ProductDTO, error) {
	products, err := s.repository.GetProducts()
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		return nil, fmt.Errorf("failed to fetch products: %w", err)
	}
	productDTOs := make([]ProductDTO, len(products))

	for i, item := range products {
		productDTOs[i] = ProductDTO{
			ID:   item.ID,
			Name: item.Name,
		}
	}

	return productDTOs, nil
}

func (s *ProductService) GetProductByID(id string) (*ProductDTO, error) {
	product, err := s.repository.GetProductByID(id)
	if err != nil {
		log.Printf("Error fetching product by ID %s: %v", id, err)
		return nil, fmt.Errorf("failed to fetch product by ID %s: %w", id, err)
	}

	return &ProductDTO{
		ID:   product.ID,
		Name: product.Name,
	}, nil
}

func (s *ProductService) CreateProduct(product ProductDTO) (*ProductDTO, error) {
	p, err := s.repository.CreateProduct(product.Name, product.Price, product.Stock)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	return &ProductDTO{
		ID:   p.ID,
		Name: p.Name,
	}, nil
}

func (s *ProductService) DeleteProductByID(id string) error {
	return s.repository.DeleteProductByID(id)
}
