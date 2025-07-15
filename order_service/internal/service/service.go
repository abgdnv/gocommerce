// Package service provides the implementation of order-related business logic.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	ordererrors "github.com/abgdnv/gocommerce/order_service/internal/errors"
	"github.com/abgdnv/gocommerce/order_service/internal/store"
	"github.com/abgdnv/gocommerce/order_service/internal/store/db"
	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	"github.com/abgdnv/gocommerce/pkg/messaging"
	"github.com/abgdnv/gocommerce/pkg/messaging/events"

	"github.com/google/uuid"
)

// OrderService defines the methods for managing orders.
// It abstracts the underlying business logic and data access.
type OrderService interface {
	// FindByID retrieves a single order by its unique identifier.
	// Returns ErrOrderNotFound if no order exists with the given ID.
	FindByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*OrderDto, error)

	// FindOrdersByUserID returns all available orders for a specific user.
	// Returns an empty slice if no orders exist.
	FindOrdersByUserID(ctx context.Context, userID uuid.UUID, offset, limit int32) (*[]OrderDto, error)

	// Create adds a new order to the system.
	// Returns error if the order cannot be created.
	Create(ctx context.Context, order OrderCreateDto) (*OrderDto, error)

	// Update modifies an existing order's details.
	// Returns ErrOrderNotFound if no order exists with the given ID and version.
	Update(ctx context.Context, userID uuid.UUID, order OrderUpdateDto) (*OrderDto, error)
}

// Service implements OrderService and provides methods to manage orders.
type Service struct {
	orderStore    store.OrderStore
	productClient pb.ProductServiceClient
	publisher     messaging.Publisher
}

// NewService creates a new instance of OrderService with the provided orderStore.
func NewService(orderStore store.OrderStore, productClient pb.ProductServiceClient, publisher messaging.Publisher) *Service {
	return &Service{
		orderStore:    orderStore,
		productClient: productClient,
		publisher:     publisher,
	}
}

// OrderDto represents the data transfer object for an order.
// Version is read-only and used for optimistic concurrency control.
type OrderDto struct {
	ID        uuid.UUID      `json:"id"`
	UserID    uuid.UUID      `json:"user_id" validate:"required"`
	Status    string         `json:"status"`
	Version   int32          `json:"version" validate:"required,min=1"`
	CreatedAt string         `json:"created_at"`
	Items     []OrderItemDto `json:"items,omitempty" validate:"required,gt=0,dive"`
}

type OrderItemDto struct {
	ID           uuid.UUID `json:"id"`
	OrderID      uuid.UUID `json:"order_id" validate:"required"`
	ProductID    uuid.UUID `json:"product_id" validate:"required"`
	Quantity     int32     `json:"quantity" validate:"required,min=1"`
	PricePerItem int64     `json:"price_per_item" validate:"required,min=0"`
	Price        int64     `json:"price" validate:"required,min=0"`
	Version      int32     `json:"version" validate:"required,min=1"`
	CreatedAt    string    `json:"created_at"`
}

// OrderCreateDto represents the data transfer object for creating a new order.
type OrderCreateDto struct {
	UserID uuid.UUID            `json:"user_id" validate:"required"`
	Status string               `json:"status"  validate:"required"`
	Items  []OrderItemCreateDto `json:"items"   validate:"required,gt=0,dive"`
}

// OrderItemCreateDto represents the data transfer object for creating a new order item.
type OrderItemCreateDto struct {
	ProductID    uuid.UUID `json:"product_id" validate:"required"`
	Quantity     int32     `json:"quantity" validate:"required,min=1"`
	PricePerItem int64     `json:"price_per_item" validate:"required,min=0"`
	Price        int64     `json:"price" validate:"required,min=0"`
}

// OrderUpdateDto represents the data transfer object for updating an existing order.
type OrderUpdateDto struct {
	ID      uuid.UUID `json:"id" validate:"required"`
	Status  string    `json:"status"  validate:"required"`
	Version int32     `json:"version" validate:"required,min=1"`
}

// FindByID retrieves an order by its ID and returns it as a OrderDto.
// Returns ErrOrderNotFound if no order exists with the given ID.
func (s *Service) FindByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*OrderDto, error) {
	order, items, err := s.orderStore.FindByID(ctx, id)
	if err != nil {
		return nil, err
	} else if order != nil && order.UserID != userID {
		return nil, ordererrors.ErrAccessDenied
	}

	return toDto(order, items), nil
}

// FindOrdersByUserID retrieves a list of all orders and returns them as OrderDtos.
// Returns an empty slice if no orders exist or error if the retrieval fails.
func (s *Service) FindOrdersByUserID(ctx context.Context, userID uuid.UUID, offset, limit int32) (*[]OrderDto, error) {
	orders, err := s.orderStore.FindOrdersByUserID(ctx, &db.FindOrdersByUserIDParams{UserID: userID, Offset: offset, Limit: limit})
	if err != nil {
		return nil, err
	}
	OrderDtos := make([]OrderDto, len(*orders))

	for i, item := range *orders {
		OrderDtos[i] = *toDto(&item, nil)
	}

	return &OrderDtos, nil
}

// Create creates a new order and returns it as a OrderDto.
// Returns an error if the order cannot be created.
func (s *Service) Create(ctx context.Context, order OrderCreateDto) (*OrderDto, error) {

	orderParams := db.CreateOrderParams{
		UserID: order.UserID,
		Status: order.Status,
	}

	var totalPrice int64
	// Check if the products exist and has sufficient stock.
	orderItems := make([]db.CreateOrderItemParams, 0, len(order.Items))
	for _, item := range order.Items {
		slog.Info("Checking product stock", slog.String("ProductID", item.ProductID.String()))
		productResp, err := s.productClient.GetProduct(ctx, &pb.GetProductRequest{Id: item.ProductID.String()})
		if err != nil {
			slog.ErrorContext(ctx, "Failed to get product info from Product service", "error", err)
			return nil, err
		}
		stockQuantity := productResp.GetProduct().GetStockQuantity()
		if stockQuantity < item.Quantity {
			message := fmt.Sprintf("product %s. Available: %d, Requested: %d", item.ProductID.String(), stockQuantity, item.Quantity)
			slog.WarnContext(ctx, fmt.Sprintf("Insufficient stock for %s", message))
			return nil, fmt.Errorf("%s: %w", message, ordererrors.ErrInsufficientStock)
		}
		orderItems = append(orderItems, db.CreateOrderItemParams{
			OrderID:      order.UserID,
			ProductID:    item.ProductID,
			Quantity:     item.Quantity,
			PricePerItem: item.PricePerItem,
			Price:        item.Price,
		})
		totalPrice += item.Price * int64(item.Quantity)

	}

	createOrder, items, err := s.orderStore.CreateOrder(ctx, &orderParams, &orderItems)
	if err != nil {
		return nil, err
	}

	event := events.OrderCreatedEvent{
		OrderID:    createOrder.ID,
		UserID:     createOrder.UserID,
		TotalPrice: totalPrice,
		CreatedAt:  *createOrder.CreatedAt,
	}
	err = s.publisher.Publish(ctx, event)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to publish OrderCreatedEvent", "error", err)
	}

	return toDto(createOrder, items), nil
}

// Update modifies an existing order's details and returns the updated order as a OrderDto.
// Returns ErrOrderNotFound if no order exists with the given ID and version.
func (s *Service) Update(ctx context.Context, userID uuid.UUID, updateDto OrderUpdateDto) (*OrderDto, error) {

	// Validate that the order exists and the user has access to it
	order, _, err := s.orderStore.FindByID(ctx, updateDto.ID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ordererrors.ErrAccessDenied
	}

	updated, err := s.orderStore.Update(ctx, &db.UpdateOrderParams{ID: updateDto.ID, Status: updateDto.Status, Version: updateDto.Version})
	if err != nil {
		return nil, err
	}

	return toDto(updated, nil), nil
}

// toDto converts a store.Order to a OrderDto.
func toDto(order *db.Order, items *[]db.OrderItem) *OrderDto {
	if order == nil {
		return nil
	}

	var itemsDto []OrderItemDto
	if items != nil {
		itemsDto = make([]OrderItemDto, 0, len(*items))
		for _, item := range *items {
			itemsDto = append(itemsDto, OrderItemDto{
				ID:           item.ID,
				OrderID:      item.OrderID,
				ProductID:    item.ProductID,
				Quantity:     item.Quantity,
				PricePerItem: item.PricePerItem,
				Price:        item.Price,
				Version:      item.Version,
				CreatedAt:    item.CreatedAt.Format(time.RFC3339),
			})
		}
	}

	return &OrderDto{
		ID:        order.ID,
		UserID:    order.UserID,
		Status:    order.Status,
		Version:   order.Version,
		CreatedAt: order.CreatedAt.Format(time.RFC3339),
		Items:     itemsDto,
	}
}
