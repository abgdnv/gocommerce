package rest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ordererrors "github.com/abgdnv/gocommerce/order_service/internal/errors"
	"github.com/abgdnv/gocommerce/order_service/internal/service"
	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	"github.com/abgdnv/gocommerce/pkg/web"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockOrderService is a mock implementation of the OrderService interface
type mockOrderService struct {
	order  *service.OrderDto
	orders []service.OrderDto
	error  error
}

func (m *mockOrderService) FindByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*service.OrderDto, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.order, nil
}

func (m *mockOrderService) FindOrdersByUserID(_ context.Context, _ uuid.UUID, _, _ int32) (*[]service.OrderDto, error) {
	if m.error != nil {
		return nil, m.error
	}
	return &m.orders, nil
}

func (m *mockOrderService) Create(_ context.Context, _ service.OrderCreateDto) (*service.OrderDto, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.order, nil
}

func (m *mockOrderService) Update(_ context.Context, _ uuid.UUID, _ service.OrderUpdateDto) (*service.OrderDto, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.order, nil
}

type ProductServiceClientMock struct {
	productResponse *pb.GetProductResponse
	error           error
	ServerTimeout   time.Duration
}

func (p ProductServiceClientMock) GetProduct(ctx context.Context, _ *pb.GetProductRequest, _ ...grpc.CallOption) (*pb.GetProductResponse, error) {
	if p.ServerTimeout > 0 {
		timer := time.NewTimer(p.ServerTimeout)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, status.Error(codes.DeadlineExceeded, "context deadline exceeded")
		case <-timer.C:
		}
	}
	if p.error != nil {
		return nil, p.error
	}
	return p.productResponse, nil
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ValidationErrorResponse struct {
	ValidationErrors map[string]string `json:"validation_errors"`
}

// toJSON is a helper function to convert a struct to JSON string
func toJSON(t *testing.T, v interface{}) string {
	t.Helper()
	bytes, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal to JSON: %v", err)
	}
	return string(bytes)
}

func Test_OrderAPI_FindByID(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	mockUserID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	createdAt := time.Now()
	testCases := []struct {
		name         string
		mockService  mockOrderService
		orderID      string
		userID       uuid.UUID
		expectedCode int
		expectedBody string
	}{
		{
			name: "Success - order found",
			mockService: mockOrderService{
				order: &service.OrderDto{
					ID:        mockID,
					UserID:    mockUserID,
					Status:    "pending",
					Version:   1,
					CreatedAt: createdAt.Format(time.RFC3339),
					Items: []service.OrderItemDto{{
						ID:           mockID,
						OrderID:      mockID,
						ProductID:    mockID,
						Quantity:     1,
						PricePerItem: 100,
						Price:        100,
						Version:      1,
						CreatedAt:    createdAt.Format(time.RFC3339),
					}}},
				error: nil,
			},
			orderID:      mockID.String(),
			userID:       mockUserID,
			expectedCode: http.StatusOK,
			expectedBody: toJSON(t, service.OrderDto{
				ID:        mockID,
				UserID:    mockUserID,
				Status:    "pending",
				Version:   1,
				CreatedAt: createdAt.Format(time.RFC3339),
				Items: []service.OrderItemDto{{
					ID:           mockID,
					OrderID:      mockID,
					ProductID:    mockID,
					Quantity:     1,
					PricePerItem: 100,
					Price:        100,
					Version:      1,
					CreatedAt:    createdAt.Format(time.RFC3339),
				}},
			}),
		},
		{
			name: "Error - unauthorized user",
			mockService: mockOrderService{
				order: nil,
				error: ordererrors.ErrAccessDenied,
			},
			orderID:      mockID.String(),
			userID:       mockUserID,
			expectedCode: http.StatusForbidden,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Access denied to order with ID " + mockID.String(),
			}),
		},
		{
			name: "Error - invalid id",
			mockService: mockOrderService{
				order: nil,
				error: errors.New("invalid order ID"),
			},
			orderID:      "123-invalid-id",
			userID:       uuid.Nil,
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Invalid ID: 123-invalid-id",
			}),
		},
		{
			name: "Error - order not found",
			mockService: mockOrderService{
				order: nil,
				error: ordererrors.ErrOrderNotFound,
			},
			orderID:      mockID.String(),
			userID:       mockUserID,
			expectedCode: http.StatusNotFound,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Order with ID " + mockID.String() + " not found",
			}),
		},
		{
			name: "Error - service error",
			mockService: mockOrderService{
				order: nil,
				error: errors.New("service unavailable"),
			},
			orderID:      mockID.String(),
			userID:       mockUserID,
			expectedCode: http.StatusInternalServerError,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Failed to retrieve order with ID " + mockID.String(),
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewHandler(&tc.mockService, nil, 0, logger)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+tc.orderID, nil)

			if tc.userID != uuid.Nil {
				ctx := context.WithValue(context.Background(), web.UserIDKey, tc.userID.String())
				req = req.WithContext(ctx)
			}

			req.SetPathValue("id", tc.orderID)
			rr := httptest.NewRecorder()

			// when
			api.FindByID(rr, req)

			// then
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			assert.Equal(t, tc.expectedCode, rr.Code, "status code should match")
			assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "response body should match")
		})
	}

}

func Test_OrderAPI_FindOrdersByUserID(t *testing.T) {
	mockUserID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	mockOrderID1, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174002")
	mockOrderID2, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174003")
	createdAt := time.Now()
	const completed = "COMPLETED"

	testCases := []struct {
		name            string
		mockService     mockOrderService
		userID          uuid.UUID
		expectedCode    int
		expectedBody    string
		noLimit         bool
		noOffset        bool
		OffsetNotNumber bool
	}{
		{
			name: "Success - orders found",
			mockService: mockOrderService{
				orders: []service.OrderDto{
					{ID: mockOrderID1, UserID: mockUserID, Status: completed, Version: 1, CreatedAt: createdAt.Format(time.RFC3339)},
					{ID: mockOrderID2, UserID: mockUserID, Status: completed, Version: 1, CreatedAt: createdAt.Format(time.RFC3339)},
				},
				error: nil,
			},
			userID:       mockUserID,
			expectedCode: http.StatusOK,
			expectedBody: toJSON(t, []service.OrderDto{
				{ID: mockOrderID1, UserID: mockUserID, Status: completed, Version: 1, CreatedAt: createdAt.Format(time.RFC3339)},
				{ID: mockOrderID2, UserID: mockUserID, Status: completed, Version: 1, CreatedAt: createdAt.Format(time.RFC3339)},
			}),
		},
		{
			name: "Success - no orders",
			mockService: mockOrderService{
				orders: []service.OrderDto{},
				error:  nil,
			},
			userID:       mockUserID,
			expectedCode: http.StatusOK,
			expectedBody: `[]`,
		},
		{
			name: "Error - service error",
			mockService: mockOrderService{
				orders: nil,
				error:  ordererrors.ErrFailedToFindUserOrders,
			},
			userID:       mockUserID,
			expectedCode: http.StatusInternalServerError,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Failed to fetch orders",
			}),
		},
		{
			name: "Error - no limit provided",
			mockService: mockOrderService{
				orders: nil,
				error:  nil,
			},
			userID:       mockUserID,
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "limit url parameter is required",
			}),
			noLimit: true,
		},
		{
			name: "Error - no offset provided",
			mockService: mockOrderService{
				orders: nil,
				error:  nil,
			},
			userID:       mockUserID,
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "offset url parameter is required",
			}),
			noOffset: true,
		},
		{
			name: "Error - offset not a number",
			mockService: mockOrderService{
				orders: nil,
				error:  nil,
			},
			userID:       mockUserID,
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Invalid offset number: not-a-number",
			}),
			OffsetNotNumber: true,
		},
		{
			name: "Error - unauthorized user",
			mockService: mockOrderService{
				orders: nil,
				error:  ordererrors.ErrAccessDenied,
			},
			userID:       mockUserID,
			expectedCode: http.StatusForbidden,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Access denied",
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewHandler(&tc.mockService, nil, 0, logger)

			params := make([]string, 0, 2)
			if !tc.noOffset {
				if tc.OffsetNotNumber {
					params = append(params, "offset=not-a-number")
				} else {
					params = append(params, "offset=0")
				}
			}
			if !tc.noLimit {
				params = append(params, "limit=100")
			}
			target := "/api/v1/orders?" + strings.Join(params, "&")

			req := httptest.NewRequest(http.MethodGet, target, nil)

			if tc.userID != uuid.Nil {
				ctx := context.WithValue(context.Background(), web.UserIDKey, tc.userID.String())
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()

			// when
			api.FindOrdersByUserID(rr, req)

			// then
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			assert.Equal(t, tc.expectedCode, rr.Code, "status code should match")
			assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "response body should match")
		})
	}
}

func Test_OrderAPI_Create(t *testing.T) {
	mockUserID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	mockOrderID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	mockItemID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174002")

	createdAt := time.Now()

	testCases := []struct {
		name                 string
		mockService          mockOrderService
		productClient        *ProductServiceClientMock
		productClientTimeout time.Duration
		requestBody          string
		expectedCode         int
		expectedBody         string
	}{
		{
			name: "Success - order created",
			mockService: mockOrderService{
				order: &service.OrderDto{ID: mockOrderID, UserID: mockUserID, Status: "pending", Version: 1, CreatedAt: createdAt.Format(time.RFC3339),
					Items: []service.OrderItemDto{{
						ID:           mockItemID,
						OrderID:      mockOrderID,
						ProductID:    mockItemID,
						Quantity:     1,
						PricePerItem: 100,
						Price:        100,
						Version:      1,
						CreatedAt:    createdAt.Format(time.RFC3339),
					}},
				},
				error: nil,
			},
			productClient: &ProductServiceClientMock{
				productResponse: &pb.GetProductResponse{
					Product: &pb.Product{
						Id:            mockItemID.String(),
						Name:          "Test Product",
						Price:         100,
						StockQuantity: 10,
						Version:       1,
					},
				},
				error: nil,
			},
			requestBody: toJSON(t, service.OrderCreateDto{
				UserID: mockUserID,
				Status: "pending",
				Items: []service.OrderItemCreateDto{{
					ProductID:    mockItemID,
					Quantity:     1,
					PricePerItem: 100,
					Price:        100,
				}},
			}),
			expectedCode: http.StatusCreated,
			expectedBody: toJSON(t, service.OrderDto{
				ID:        mockOrderID,
				UserID:    mockUserID,
				Status:    "pending",
				Version:   1,
				CreatedAt: createdAt.Format(time.RFC3339),
				Items: []service.OrderItemDto{{
					ID:           mockItemID,
					OrderID:      mockOrderID,
					ProductID:    mockItemID,
					Quantity:     1,
					PricePerItem: 100,
					Price:        100,
					Version:      1,
					CreatedAt:    createdAt.Format(time.RFC3339),
				}},
			}),
		},
		{
			name: "Error - validation failed - invalid user_id (uuid)",
			mockService: mockOrderService{
				order: nil,
				error: nil,
			},
			requestBody:  `{"user_id":"","status":"","items":[]}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Invalid request body",
			}),
		},
		{
			name: "Error - validation failed - order status and items",
			mockService: mockOrderService{
				order: nil,
				error: nil,
			},
			requestBody: toJSON(t, service.OrderCreateDto{
				UserID: mockUserID,
				Status: "",                             // Invalid status
				Items:  []service.OrderItemCreateDto{}, // Empty items array
			}),
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ValidationErrorResponse{
				ValidationErrors: map[string]string{
					"Status": "failed on rule: required",
					"Items":  "failed on rule: gt",
				},
			}),
		},
		{
			name: "Error - validation failed - order items",
			mockService: mockOrderService{
				order: nil,
				error: nil,
			},
			requestBody: toJSON(t, service.OrderCreateDto{
				UserID: mockUserID,
				Status: "pending",
				Items: []service.OrderItemCreateDto{{
					ProductID:    mockItemID,
					Quantity:     0,    // Invalid quantity
					PricePerItem: -100, // Invalid price
					Price:        -100, // Invalid price

				}},
			}),
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ValidationErrorResponse{
				ValidationErrors: map[string]string{
					"Quantity":     "failed on rule: required",
					"PricePerItem": "failed on rule: min",
					"Price":        "failed on rule: min",
				},
			}),
		},
		{
			name: "Error - invalid json",
			mockService: mockOrderService{
				order: nil,
				error: nil,
			},
			requestBody:  `invalid json`,
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Invalid request body",
			}),
		},
		{
			name: "Error - service error",
			mockService: mockOrderService{
				order: nil,
				error: errors.New("service unavailable"),
			},
			productClient: &ProductServiceClientMock{
				productResponse: &pb.GetProductResponse{
					Product: &pb.Product{
						Id:            mockItemID.String(),
						Name:          "Test Product",
						Price:         100,
						StockQuantity: 10,
						Version:       1,
					},
				},
				error: nil,
			},
			requestBody: toJSON(t, service.OrderCreateDto{
				UserID: mockUserID,
				Status: "pending",
				Items: []service.OrderItemCreateDto{{
					ProductID:    mockItemID,
					Quantity:     1,
					PricePerItem: 100,
					Price:        100,
				}},
			}),
			expectedCode: http.StatusInternalServerError,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Failed to create order",
			}),
		},
		{
			name: "Error - stock not enough",
			productClient: &ProductServiceClientMock{
				productResponse: &pb.GetProductResponse{
					Product: &pb.Product{
						Id:            mockItemID.String(),
						Name:          "Test Product",
						Price:         100,
						StockQuantity: 1,
						Version:       1,
					},
				},
				error: nil,
			},
			requestBody: toJSON(t, service.OrderCreateDto{
				UserID: mockUserID,
				Status: "pending",
				Items: []service.OrderItemCreateDto{{
					ProductID:    mockItemID,
					Quantity:     10,
					PricePerItem: 100,
					Price:        100,
				}},
			}),
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Insufficient stock for product " + mockItemID.String() + ". Available: 1, Requested: 10",
			}),
		},
		{
			name: "Error - product service timeout",
			mockService: mockOrderService{
				order: nil,
				error: nil,
			},
			productClient: &ProductServiceClientMock{
				productResponse: &pb.GetProductResponse{},
				error:           nil,
				ServerTimeout:   3 * time.Second, // Simulate a timeout
			},
			productClientTimeout: 2 * time.Second,
			requestBody: toJSON(t, service.OrderCreateDto{
				UserID: mockUserID,
				Status: "pending",
				Items: []service.OrderItemCreateDto{{
					ProductID:    mockItemID,
					Quantity:     1,
					PricePerItem: 100,
					Price:        100,
				}},
			}),
			expectedCode: http.StatusGatewayTimeout,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "The request timed out",
			}),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewHandler(&tc.mockService, tc.productClient, tc.productClientTimeout, logger)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
			req.Body = io.NopCloser(strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			ctx := context.WithValue(context.Background(), web.UserIDKey, mockUserID.String())
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			// when
			api.Create(rr, req)
			// then
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			assert.Equal(t, tc.expectedCode, rr.Code, "status code should match")
			assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "response body should match")
		})
	}
}

func Test_OrderAPI_Update(t *testing.T) {
	mockOrderID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	mockUserID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	createdAt := time.Now()
	const pending = "PENDING"
	testCases := []struct {
		name         string
		mockService  mockOrderService
		orderID      uuid.UUID
		userID       uuid.UUID
		requestBody  string
		expectedCode int
		expectedBody string
	}{
		{
			name: "Success - order updated",
			mockService: mockOrderService{
				order: &service.OrderDto{ID: mockOrderID, UserID: mockUserID, Status: pending, Version: 2, CreatedAt: createdAt.Format(time.RFC3339)},
				error: nil,
			},
			orderID: mockOrderID,
			userID:  mockUserID,
			requestBody: toJSON(t, service.OrderUpdateDto{
				Status:  pending,
				Version: 1,
			}),
			expectedCode: http.StatusOK,
			expectedBody: toJSON(t, service.OrderDto{
				ID:        mockOrderID,
				UserID:    mockUserID,
				Status:    pending,
				Version:   2,
				CreatedAt: createdAt.Format(time.RFC3339),
			}),
		},
		{
			name: "Error - validation failed",
			mockService: mockOrderService{
				order: nil,
				error: nil,
			},
			orderID: mockOrderID,
			userID:  mockUserID,
			requestBody: toJSON(t, service.OrderUpdateDto{
				Status:  "", // Invalid status
				Version: 0,  // Invalid version
			}),
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ValidationErrorResponse{
				ValidationErrors: map[string]string{
					"Status":  "failed on rule: required",
					"Version": "failed on rule: required",
				},
			}),
		},
		{
			name: "Error - invalid json",
			mockService: mockOrderService{
				order: nil,
				error: nil,
			},
			orderID:      mockOrderID,
			userID:       mockUserID,
			requestBody:  `invalid json`,
			expectedCode: http.StatusBadRequest,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Invalid request body",
			}),
		},
		{
			name: "Error - order not found",
			mockService: mockOrderService{
				order: nil,
				error: ordererrors.ErrOrderNotFound,
			},
			orderID: mockOrderID,
			userID:  mockUserID,
			requestBody: toJSON(t, service.OrderUpdateDto{
				Status:  pending,
				Version: 1,
			}),
			expectedCode: http.StatusNotFound,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Order with ID " + mockOrderID.String() + " not found",
			}),
		},
		{
			name: "Error - service error",
			mockService: mockOrderService{
				order: nil,
				error: errors.New("service unavailable"),
			},
			orderID: mockOrderID,
			userID:  mockUserID,
			requestBody: toJSON(t, service.OrderUpdateDto{
				Status:  pending,
				Version: 1,
			}),
			expectedCode: http.StatusInternalServerError,
			expectedBody: toJSON(t, ErrorResponse{
				Error: "Failed to update order with ID " + mockOrderID.String(),
			}),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewHandler(&tc.mockService, nil, 0, logger)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+tc.orderID.String(), nil)
			req.Body = io.NopCloser(strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("id", tc.orderID.String())
			if tc.userID != uuid.Nil {
				ctx := context.WithValue(context.Background(), web.UserIDKey, tc.userID.String())
				req = req.WithContext(ctx)
			}
			rr := httptest.NewRecorder()

			// when
			api.Update(rr, req)

			// then
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			assert.Equal(t, tc.expectedCode, rr.Code, "status code should match")
			assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "response body should match")
		})
	}

}

func Test_OrderAPI_HealthCheck(t *testing.T) {
	// given
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	api := NewHandler(nil, nil, 0, logger) // No service needed for health check
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rr := httptest.NewRecorder()

	// when
	api.HealthCheck(rr, req)

	// then
	assert.Equal(t, http.StatusOK, rr.Code, "status code should be 200 OK")
	assert.Empty(t, rr.Body.String(), "response body should be empty")
}
