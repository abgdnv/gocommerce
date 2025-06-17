package repository

type ProductRepositoryContract interface {
	GetProductByID(id string) (*Product, error)
	GetProducts() ([]Product, error)
	CreateProduct(name string, price int64, stock int32) (Product, error)
	DeleteProductByID(id string) error
}
