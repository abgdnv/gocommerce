package service

type ProductServiceContract interface {
	GetProducts() ([]ProductDTO, error)
	GetProductByID(id string) (*ProductDTO, error)
	CreateProduct(product ProductDTO) (*ProductDTO, error)
	DeleteProductByID(id string) error
}
