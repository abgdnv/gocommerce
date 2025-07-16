package store

import (
	"context"
	"errors"
	"fmt"

	perrors "github.com/abgdnv/gocommerce/product_service/internal/errors"
	"github.com/abgdnv/gocommerce/product_service/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore implements ProductStore using PostgreSQL as the data store.
type PgStore struct {
	db *pgxpool.Pool
	q  *db.Queries
}

// NewPgStore creates a new instance of ProductStore using a PostgreSQL connection pool.
func NewPgStore(dbp *pgxpool.Pool) *PgStore {
	return &PgStore{
		db: dbp,
		q:  db.New(dbp),
	}
}

// FindByID retrieves a product by its unique identifier.
// Returns ErrProductNotFound if no product exists with the given ID.
func (p *PgStore) FindByID(ctx context.Context, id uuid.UUID) (*db.Product, error) {
	product, err := p.q.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, perrors.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to find product by ID: %w", err)
	}
	return &product, nil
}

// FindByIDs retrieves products by IDs
// It returns a slice of products, which may be empty if no products exist.
func (p *PgStore) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]db.Product, error) {
	products, err := p.q.FindByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to find all products: %w", err)
	}
	return products, nil
}

// FindAll retrieves all available products with pagination support.
// It returns a slice of products, which may be empty if no products exist.
func (p *PgStore) FindAll(ctx context.Context, offset, limit int32) ([]db.Product, error) {
	products, err := p.q.FindAll(ctx, db.FindAllParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, fmt.Errorf("failed to find all products: %w", err)
	}
	return products, nil
}

// Create adds a new product to the system.
// Returns an error if the product cannot be created.
func (p *PgStore) Create(ctx context.Context, name string, price int64, stock int32) (*db.Product, error) {
	product, err := p.q.Create(ctx, db.CreateParams{
		Name:          name,
		Price:         price,
		StockQuantity: stock,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}
	return &product, nil
}

// Update modifies an existing product's details.
// Returns ErrProductNotFound if no product exists with the given ID and version.
func (p *PgStore) Update(ctx context.Context, id uuid.UUID, name string, price int64, stock int32, version int32) (*db.Product, error) {
	product, err := p.q.Update(ctx, db.UpdateParams{
		ID:            id,
		Name:          name,
		Price:         price,
		StockQuantity: stock,
		Version:       version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, perrors.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to update product: %w", err)
	}
	return &product, nil
}

// UpdateStock adjusts the stock quantity of a product.
// Returns ErrProductNotFound if no product exists with the given ID and version.
func (p *PgStore) UpdateStock(ctx context.Context, id uuid.UUID, stock int32, version int32) (*db.Product, error) {
	product, err := p.q.UpdateStock(ctx, db.UpdateStockParams{
		ID:            id,
		StockQuantity: stock,
		Version:       version,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, perrors.ErrProductNotFound
		}
		return nil, fmt.Errorf("failed to update product stock: %w", err)
	}
	return &product, nil
}

// DeleteByID removes a product by its unique identifier.
// Returns ErrProductNotFound if no product exists with the given ID and version.
func (p *PgStore) DeleteByID(ctx context.Context, id uuid.UUID, version int32) error {
	count, err := p.q.Delete(ctx, db.DeleteParams{
		ID:      id,
		Version: version,
	})
	if err != nil {
		return fmt.Errorf("failed to delete product by ID: %w", err)
	}
	if count == 0 {
		return perrors.ErrProductNotFound
	}
	return nil
}
