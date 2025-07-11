package store

import (
	"context"
	"errors"

	ordererrors "github.com/abgdnv/gocommerce/order_service/internal/errors"
	"github.com/abgdnv/gocommerce/order_service/internal/store/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

func (p *PgStore) FindByID(ctx context.Context, id uuid.UUID) (*db.Order, *[]db.OrderItem, error) {
	var order *db.Order
	var orderItems *[]db.OrderItem

	// Use transaction to ensure atomicity and consistency
	txErr := p.withTransaction(ctx, func(qtx *db.Queries) error {
		o, err := qtx.FindOrderByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ordererrors.ErrOrderNotFound
			}
			return ordererrors.ErrFailedToFindOrder
		}
		i, err := qtx.FindOrderItemsByOrderID(ctx, id)
		if err != nil {
			return ordererrors.ErrFailedToFindOrderItems
		}
		order = &o
		orderItems = &i
		return nil
	})

	if txErr != nil {
		return nil, nil, txErr
	}

	return order, orderItems, nil
}

func (p *PgStore) FindOrdersByUserID(ctx context.Context, params *db.FindOrdersByUserIDParams) (*[]db.Order, error) {

	// No need for transaction here as we are making just one query to fetch orders
	orders, err := p.q.FindOrdersByUserID(ctx, *params)
	if err != nil {
		return nil, ordererrors.ErrFailedToFindUserOrders
	}

	return &orders, nil
}

func (p *PgStore) CreateOrder(ctx context.Context, orderParams *db.CreateOrderParams, items *[]db.CreateOrderItemParams) (*db.Order, *[]db.OrderItem, error) {
	var createdOrder *db.Order
	var createdItems *[]db.OrderItem

	txErr := p.withTransaction(ctx, func(qtx *db.Queries) error {
		order, err := qtx.CreateOrder(ctx, *orderParams)
		if err != nil {
			return ordererrors.ErrCreateOrder
		}
		orderItems := make([]db.OrderItem, 0, len(*items))
		for _, item := range *items {
			item.OrderID = order.ID
			orderItem, err := qtx.CreateOrderItem(ctx, item)
			if err != nil {
				return ordererrors.ErrCreateOrderItem
			}
			orderItems = append(orderItems, orderItem)
		}
		createdOrder = &order
		createdItems = &orderItems
		return nil
	})

	if txErr != nil {
		return nil, nil, txErr
	}

	return createdOrder, createdItems, nil
}

func (p *PgStore) Update(ctx context.Context, params *db.UpdateOrderParams) (*db.Order, error) {
	var order db.Order

	txErr := p.withTransaction(ctx, func(qtx *db.Queries) error {
		var err error
		order, err = qtx.UpdateOrder(ctx, *params)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Check if the order exists, or it's an optimistic lock error.
				order, err = qtx.FindOrderByID(ctx, params.ID)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						return ordererrors.ErrOrderNotFound
					}
				} else {
					return ordererrors.ErrOptimisticLock
				}

			}
			return ordererrors.ErrUpdateOrder
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	return &order, nil
}

func (p *PgStore) withTransaction(ctx context.Context, fn func(qtx *db.Queries) error) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return ordererrors.ErrTransactionBegin
	}
	qtx := p.q.WithTx(tx)

	err = fn(qtx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			return ordererrors.ErrTransactionRollback
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return ordererrors.ErrTransactionCommit
	}

	return nil
}
