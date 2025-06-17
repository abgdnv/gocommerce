// Package store provides an interface for product storage operations.
package store

// Store is an interface for product storage operations.
type Store interface {
	GetProductByID(id string) (*Product, error)
	GetProducts() ([]Product, error)
	CreateProduct(name string, price int64, stock int32) (Product, error)
	DeleteProductByID(id string) error
}
