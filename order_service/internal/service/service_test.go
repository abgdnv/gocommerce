package service

import (
	"context"
	"testing"
	"time"

	ordererrors "github.com/abgdnv/gocommerce/order_service/internal/errors"
	"github.com/abgdnv/gocommerce/order_service/internal/store/db"
	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockOrderStore is a mock implementation of the OrderStore interface
type mockOrderStore struct {
	orders      *[]db.Order
	order       *db.Order
	items       *[]db.OrderItem
	error       error
	updateOrder *db.Order
	updateError error
}

func (m *mockOrderStore) FindByID(_ context.Context, _ uuid.UUID) (*db.Order, *[]db.OrderItem, error) {
	if m.error != nil {
		return nil, nil, m.error
	}
	return m.order, m.items, nil
}

func (m *mockOrderStore) FindOrdersByUserID(_ context.Context, _ *db.FindOrdersByUserIDParams) (*[]db.Order, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.orders, nil
}

func (m *mockOrderStore) CreateOrder(_ context.Context, _ *db.CreateOrderParams, _ *[]db.CreateOrderItemParams) (*db.Order, *[]db.OrderItem, error) {
	if m.error != nil {
		return nil, nil, m.error
	}
	return m.order, m.items, nil
}

func (m *mockOrderStore) Update(_ context.Context, _ *db.UpdateOrderParams) (*db.Order, error) {
	if m.updateError != nil {
		return nil, m.updateError
	}
	return m.updateOrder, nil
}

type ProductServiceClientMock struct {
	productResponse *pb.GetProductResponse
	error           error
	ServerTimeout   time.Duration
}

var errContextDeadlineExceeded = status.Error(codes.DeadlineExceeded, "context deadline exceeded")

func (p ProductServiceClientMock) GetProduct(ctx context.Context, _ *pb.GetProductRequest, _ ...grpc.CallOption) (*pb.GetProductResponse, error) {
	if p.ServerTimeout > 0 {
		timer := time.NewTimer(p.ServerTimeout)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, errContextDeadlineExceeded
		case <-timer.C:
		}
	}
	if p.error != nil {
		return nil, p.error
	}
	return p.productResponse, nil
}

func assertEqualOrderDto(t *testing.T, expected, actual *OrderDto) {
	t.Helper()
	if expected == nil || actual == nil {
		assert.Equal(t, expected, actual)
		return
	}
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.UserID, actual.UserID)
	assert.Equal(t, expected.Status, actual.Status)
	assert.Equal(t, expected.Version, actual.Version)
	assert.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Len(t, actual.Items, len(expected.Items))
	for i := range expected.Items {
		assertEqualOrderItemDto(t, &expected.Items[i], &actual.Items[i])
	}
}

func assertEqualOrderItemDto(t *testing.T, expected, actual *OrderItemDto) {
	t.Helper()
	if expected == nil || actual == nil {
		assert.Equal(t, expected, actual)
		return
	}
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.OrderID, actual.OrderID)
	assert.Equal(t, expected.ProductID, actual.ProductID)
	assert.Equal(t, expected.Quantity, actual.Quantity)
	assert.Equal(t, expected.PricePerItem, actual.PricePerItem)
	assert.Equal(t, expected.Price, actual.Price)
	assert.Equal(t, expected.Version, actual.Version)
	assert.Equal(t, expected.CreatedAt, actual.CreatedAt)

}

func Test_OrderService_FindByID(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	mockUserID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	mockProductID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174002")
	mockOrderItemID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174003")

	createdAt := time.Now()
	testCases := []struct {
		name        string
		mockStore   *mockOrderStore
		orderID     uuid.UUID
		userID      uuid.UUID
		expected    *OrderDto
		expectError error
	}{
		{
			name: "Success - order found",
			mockStore: &mockOrderStore{
				order: &db.Order{ID: mockID, UserID: mockUserID, Status: "PENDING", Version: 1, CreatedAt: &createdAt},
				items: &[]db.OrderItem{{ID: mockOrderItemID, OrderID: mockID, ProductID: mockProductID, Quantity: 1, Price: 100, CreatedAt: &createdAt}},
				error: nil,
			},
			orderID: mockID,
			userID:  mockUserID,
			expected: &OrderDto{
				ID:        mockID,
				UserID:    mockUserID,
				Status:    "PENDING",
				Version:   1,
				CreatedAt: createdAt.Format(time.RFC3339),
				Items: []OrderItemDto{{
					ID:        mockOrderItemID,
					OrderID:   mockID,
					ProductID: mockProductID,
					Quantity:  1, Price: 100,
					CreatedAt: createdAt.Format(time.RFC3339),
				}}},
			expectError: nil,
		},
		{
			name: "Error - order not found",
			mockStore: &mockOrderStore{
				error: ordererrors.ErrOrderNotFound,
			},
			orderID:     mockID,
			userID:      mockUserID,
			expected:    nil,
			expectError: ordererrors.ErrOrderNotFound,
		},
		{
			name: "Error - access denied",
			mockStore: &mockOrderStore{
				order: &db.Order{ID: mockID, UserID: uuid.New(), Status: "PENDING", Version: 1, CreatedAt: &createdAt},
				error: nil,
			},
			orderID:     mockID,
			userID:      mockUserID,
			expected:    nil,
			expectError: ordererrors.ErrAccessDenied,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore, nil, 0)
			// when
			found, err := service.FindByID(context.Background(), tc.userID, tc.orderID)
			// then
			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)
				assert.Nil(t, found)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, found)
			assertEqualOrderDto(t, tc.expected, found)
		})
	}
}

func Test_OrderService_FindOrdersByUserID(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	mockUserID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	createdAt := time.Now()
	testCases := []struct {
		name         string
		mockStore    *mockOrderStore
		userID       uuid.UUID
		expectedList []OrderDto
		expectError  error
	}{
		{
			name: "Success - orders found",
			mockStore: &mockOrderStore{
				orders: &[]db.Order{{ID: mockID, UserID: mockUserID, Status: "PENDING", Version: 1, CreatedAt: &createdAt}},
				error:  nil,
			},
			userID: mockUserID,
			expectedList: []OrderDto{
				{
					ID:        mockID,
					UserID:    mockUserID,
					Status:    "PENDING",
					Version:   1,
					Items:     nil,
					CreatedAt: createdAt.Format(time.RFC3339),
				}},
			expectError: nil,
		},
		{
			name:         "Success - no orders",
			mockStore:    &mockOrderStore{orders: &[]db.Order{}, error: nil},
			userID:       mockUserID,
			expectedList: []OrderDto{},
			expectError:  nil,
		},
		{
			name: "Error - store error",
			mockStore: &mockOrderStore{
				error: ordererrors.ErrFailedToFindUserOrders,
			},
			userID:       mockUserID,
			expectedList: nil,
			expectError:  ordererrors.ErrFailedToFindUserOrders,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore, nil, 0)
			// when
			found, err := service.FindOrdersByUserID(context.Background(), tc.userID, 0, 10)
			// then
			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)
				assert.Nil(t, found)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedList, *found)
		})
	}
}

func Test_OrderService_Create(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	userID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	ProductID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174002")
	OrderItemID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174003")

	createdAt := time.Now()
	testCases := []struct {
		name                 string
		mockStore            *mockOrderStore
		productClient        *ProductServiceClientMock
		productClientTimeout time.Duration
		order                OrderCreateDto
		expected             *OrderDto
		expectError          error
	}{
		{
			name: "Success - order created",
			mockStore: &mockOrderStore{
				order: &db.Order{ID: mockID, UserID: userID, Status: "PENDING", Version: 1, CreatedAt: &createdAt},
				items: &[]db.OrderItem{{ID: OrderItemID, OrderID: mockID, ProductID: ProductID, Quantity: 1, Price: 100, CreatedAt: &createdAt}},
				error: nil,
			},
			productClient: &ProductServiceClientMock{
				productResponse: &pb.GetProductResponse{
					Product: &pb.Product{
						Id:            ProductID.String(),
						Name:          "Test Product",
						Price:         100,
						StockQuantity: 10,
						Version:       1,
					},
				},
				error: nil,
			},
			order: OrderCreateDto{UserID: userID, Status: "PENDING", Items: []OrderItemCreateDto{{ProductID: ProductID, Quantity: 1, Price: 100}}},
			expected: &OrderDto{ID: mockID, UserID: userID, Status: "PENDING", Version: 1, CreatedAt: createdAt.Format(time.RFC3339),
				Items: []OrderItemDto{{ID: OrderItemID, OrderID: mockID, ProductID: ProductID, Quantity: 1, Price: 100, CreatedAt: createdAt.Format(time.RFC3339)}}},
			expectError: nil,
		},
		{
			name: "Error - store error",
			mockStore: &mockOrderStore{
				error: ordererrors.ErrCreateOrder,
			},
			productClient: &ProductServiceClientMock{
				productResponse: &pb.GetProductResponse{
					Product: &pb.Product{
						Id:            ProductID.String(),
						Name:          "Test Product",
						Price:         100,
						StockQuantity: 1,
						Version:       1,
					},
				},
				error: nil,
			},
			order:       OrderCreateDto{UserID: userID, Status: "PENDING", Items: []OrderItemCreateDto{{ProductID: ProductID, Quantity: 1, Price: 100}}},
			expected:    nil,
			expectError: ordererrors.ErrCreateOrder,
		},
		{
			name: "Error - insufficient stock",
			productClient: &ProductServiceClientMock{
				productResponse: &pb.GetProductResponse{
					Product: &pb.Product{
						Id:            ProductID.String(),
						Name:          "Test Product",
						Price:         100,
						StockQuantity: 1,
						Version:       1,
					},
				},
				error: nil,
			},
			order:       OrderCreateDto{UserID: userID, Status: "PENDING", Items: []OrderItemCreateDto{{ProductID: ProductID, Quantity: 10, Price: 100}}},
			expectError: ordererrors.ErrInsufficientStock,
		},
		{
			name: "Error - product service timeout",
			productClient: &ProductServiceClientMock{
				ServerTimeout: 3 * time.Second,
			},
			productClientTimeout: 2 * time.Second,
			order:                OrderCreateDto{UserID: userID, Status: "PENDING", Items: []OrderItemCreateDto{{ProductID: ProductID, Quantity: 10, Price: 100}}},
			expectError:          errContextDeadlineExceeded,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore, tc.productClient, tc.productClientTimeout)
			// when
			created, err := service.Create(context.Background(), tc.order)
			// then
			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)
				assert.Nil(t, created)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, created)
		})
	}
}

func Test_OrderService_Update(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	mockUserID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	createdAt := time.Now()

	testCases := []struct {
		name        string
		mockStore   *mockOrderStore
		order       OrderUpdateDto
		expected    *OrderDto
		expectError error
	}{
		{
			name: "Success - order updated",
			mockStore: &mockOrderStore{
				order:       &db.Order{ID: mockID, UserID: mockUserID, Status: "PENDING", Version: 1, CreatedAt: &createdAt},
				error:       nil,
				updateOrder: &db.Order{ID: mockID, UserID: mockUserID, Status: "PENDING", Version: 2, CreatedAt: &createdAt},
				updateError: nil,
			},
			order:       OrderUpdateDto{ID: mockID, Status: "PENDING", Version: 1},
			expected:    &OrderDto{ID: mockID, UserID: mockUserID, Status: "PENDING", Version: 2, CreatedAt: createdAt.Format(time.RFC3339)},
			expectError: nil,
		},
		{
			name: "Error - order not found",
			mockStore: &mockOrderStore{
				error: ordererrors.ErrOrderNotFound,
			},
			order:       OrderUpdateDto{ID: mockID, Status: "PENDING", Version: 1},
			expected:    nil,
			expectError: ordererrors.ErrOrderNotFound,
		},
		{
			name: "Error - store error",
			mockStore: &mockOrderStore{
				order:       &db.Order{ID: mockID, UserID: mockUserID, Status: "PENDING", Version: 1, CreatedAt: &createdAt},
				error:       nil,
				updateOrder: nil,
				updateError: ordererrors.ErrUpdateOrder,
			},
			order:       OrderUpdateDto{ID: mockID, Status: "PENDING", Version: 1},
			expected:    nil,
			expectError: ordererrors.ErrUpdateOrder,
		},
		{
			name: "Error - access denied",
			mockStore: &mockOrderStore{
				order: &db.Order{ID: mockID, UserID: uuid.New(), Status: "PENDING", Version: 1, CreatedAt: &createdAt},
				error: nil,
			},
			order:       OrderUpdateDto{ID: mockID, Status: "PENDING", Version: 1},
			expected:    nil,
			expectError: ordererrors.ErrAccessDenied,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore, nil, 0)
			// when
			updated, err := service.Update(context.Background(), mockUserID, tc.order)
			// then
			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)
				assert.Nil(t, updated)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, updated)
		})
	}
}

func Test_toDto(t *testing.T) {
	// given
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	mockUserID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	mockProductID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174002")
	mockOrderItemID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174003")

	createdAt := time.Now()
	testCases := []struct {
		name     string
		order    *db.Order
		items    *[]db.OrderItem
		expected *OrderDto
	}{
		{
			name:     "Nil order",
			order:    nil,
			items:    nil,
			expected: nil,
		},
		{
			name: "Order with items",
			order: &db.Order{
				ID:        mockID,
				UserID:    mockUserID,
				Status:    "PENDING",
				Version:   1,
				CreatedAt: &createdAt,
			},
			items: &[]db.OrderItem{
				{
					ID:           mockOrderItemID,
					OrderID:      mockID,
					ProductID:    mockProductID,
					Quantity:     2,
					PricePerItem: 50,
					Price:        100,
					CreatedAt:    &createdAt,
					Version:      1,
				},
			},
			expected: &OrderDto{
				ID:        mockID,
				UserID:    mockUserID,
				Status:    "PENDING",
				Version:   1,
				CreatedAt: createdAt.Format(time.RFC3339),
				Items: []OrderItemDto{
					{
						ID:           mockOrderItemID,
						OrderID:      mockID,
						ProductID:    mockProductID,
						Quantity:     2,
						PricePerItem: 50,
						Price:        100,
						CreatedAt:    createdAt.Format(time.RFC3339),
						Version:      1,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// when
			result := toDto(tc.order, tc.items)
			// then
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assertEqualOrderDto(t, tc.expected, result)
			}
		})
	}
}
