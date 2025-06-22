package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	producterrors "github.com/abgdnv/gocommerce/internal/product/errors"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockProductService is a mock implementation of the ProductService interface
type mockProductService struct {
	product  *service.ProductDto
	products []service.ProductDto
	error    error
}

// Simulate finding a product by ID
func (m mockProductService) FindByID(_ context.Context, _ uuid.UUID) (*service.ProductDto, error) {
	return m.product, m.error
}

func (m mockProductService) FindAll(_ context.Context, _, _ int32) (*[]service.ProductDto, error) {
	return &m.products, m.error
}

// Simulate creating a product
func (m mockProductService) Create(_ context.Context, _ service.ProductCreateDto) (*service.ProductDto, error) {
	return m.product, m.error
}

// Simulate updating a product
func (m mockProductService) Update(_ context.Context, _ service.ProductDto) (*service.ProductDto, error) {
	return m.product, m.error
}

// Simulate updating stock for a product
func (m mockProductService) UpdateStock(_ context.Context, _ uuid.UUID, _ int32, _ int32) (*service.ProductDto, error) {
	return m.product, m.error
}

// Simulate deleting a product by ID
func (m mockProductService) DeleteByID(_ context.Context, _ uuid.UUID, _ int32) error {
	return m.error
}

func Test_ProductAPI_FindByID(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name         string
		mockService  mockProductService
		productID    string
		expectedCode int
		expectedBody string
	}{
		{
			name: "Success - product found",
			mockService: mockProductService{
				product: &service.ProductDto{ID: mockID.String(), Name: "Product 1", Price: 100, Stock: 10, Version: 1},
				error:   nil,
			},
			productID:    mockID.String(),
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"` + mockID.String() + `","name":"Product 1","price":100,"stock":10, "version":1}`,
		},
		{
			name: "Error - invalid id",
			mockService: mockProductService{
				product: nil,
				error:   errors.New("invalid product ID"),
			},
			productID:    "123-invalid-id",
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"Invalid product ID: 123-invalid-id"}`,
		},
		{
			name: "Error - product not found",
			mockService: mockProductService{
				product: nil,
				error:   producterrors.ErrProductNotFound,
			},
			productID:    mockID.String(),
			expectedCode: http.StatusNotFound,
			expectedBody: `{"error":"Product with ID ` + mockID.String() + ` not found"}`,
		},
		{
			name: "Error - service error",
			mockService: mockProductService{
				product: nil,
				error:   errors.New("service unavailable"),
			},
			productID:    mockID.String(),
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to retrieve product with ID ` + mockID.String() + `"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewAPI(&tc.mockService, logger)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+tc.productID, nil)
			req.SetPathValue("id", tc.productID)
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

func Test_ProductAPI_FindAll(t *testing.T) {
	ErrServiceUnavailable := errors.New("service unavailable")
	testCases := []struct {
		name            string
		mockService     mockProductService
		expectedCode    int
		expectedBody    string
		noLimit         bool
		noOffset        bool
		OffsetNotNumber bool
	}{
		{
			name: "Success - products found",
			mockService: mockProductService{
				products: []service.ProductDto{
					{ID: "1", Name: "Product 1", Price: 100, Stock: 10, Version: 1},
					{ID: "2", Name: "Product 2", Price: 200, Stock: 20, Version: 1},
				},
				error: nil,
			},
			expectedCode: http.StatusOK,
			expectedBody: `[{"id":"1","name":"Product 1","price":100,"stock":10,"version":1},{"id":"2","name":"Product 2","price":200,"stock":20,"version":1}]`,
		},
		{
			name: "Success - no products",
			mockService: mockProductService{
				products: []service.ProductDto{},
				error:    nil,
			},
			expectedCode: http.StatusOK,
			expectedBody: `[]`,
		},
		{
			name: "Error - service error",
			mockService: mockProductService{
				products: nil,
				error:    ErrServiceUnavailable,
			},
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to fetch products"}`,
		},
		{
			name: "Error - no limit provided",
			mockService: mockProductService{
				products: nil,
				error:    nil,
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"limit url parameter is required"}`,
			noLimit:      true,
		},
		{
			name: "Error - no offset provided",
			mockService: mockProductService{
				products: nil,
				error:    nil,
			},
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"offset url parameter is required"}`,
			noOffset:     true,
		},
		{
			name: "Error - offset not a number",
			mockService: mockProductService{
				products: nil,
				error:    nil,
			},
			expectedCode:    http.StatusBadRequest,
			expectedBody:    `{"error":"Invalid offset number: not-a-number"}`,
			OffsetNotNumber: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewAPI(&tc.mockService, logger)

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
			target := "/api/v1/products?" + strings.Join(params, "&")

			req := httptest.NewRequest(http.MethodGet, target, nil)
			rr := httptest.NewRecorder()

			// when
			api.FindAll(rr, req)

			// then
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			assert.Equal(t, tc.expectedCode, rr.Code, "status code should match")
			assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "response body should match")
		})
	}
}

func Test_ProductAPI_Create(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name         string
		mockService  mockProductService
		requestBody  string
		expectedCode int
		expectedBody string
	}{
		{
			name: "Success - product created",
			mockService: mockProductService{
				product: &service.ProductDto{ID: mockID.String(), Name: "New Product", Price: 150, Stock: 5, Version: 1},
				error:   nil,
			},
			requestBody:  `{"name":"New Product","price":150,"stock":5}`,
			expectedCode: http.StatusCreated,
			expectedBody: `{"id":"` + mockID.String() + `","name":"New Product","price":150,"stock":5, "version":1}`,
		},
		{
			name: "Error - validation failed",
			mockService: mockProductService{
				product: nil,
				error:   nil,
			},
			requestBody:  `{"name":"","price":-100,"stock":-5}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"validation_errors":{"Name":"failed on rule: required","Price":"failed on rule: min","Stock":"failed on rule: min"}}`,
		},
		{
			name: "Error - invalid json",
			mockService: mockProductService{
				product: nil,
				error:   nil,
			},
			requestBody:  `invalid json`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"Invalid request body"}`,
		},
		{
			name: "Error - service error",
			mockService: mockProductService{
				product: nil,
				error:   errors.New("service unavailable"),
			},
			requestBody:  `{"name":"Another Product","price":200,"stock":10}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to create product"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewAPI(&tc.mockService, logger)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/products", nil)
			req.Body = io.NopCloser(strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
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

func Test_ProductAPI_Update(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name         string
		mockService  mockProductService
		productID    string
		requestBody  string
		expectedCode int
		expectedBody string
	}{
		{
			name: "Success - product updated",
			mockService: mockProductService{
				product: &service.ProductDto{ID: mockID.String(), Name: "Updated Product", Price: 200, Stock: 15, Version: 1},
				error:   nil,
			},
			productID:    mockID.String(),
			requestBody:  `{"name":"Updated Product","price":200,"stock":15,"version":1}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"` + mockID.String() + `","name":"Updated Product","price":200,"stock":15, "version":1}`,
		},
		{
			name: "Error - validation failed",
			mockService: mockProductService{
				product: nil,
				error:   nil,
			},
			productID:    mockID.String(),
			requestBody:  `{"name":"","price":-100,"stock":-5,"version":1}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"validation_errors":{"Name":"failed on rule: required","Price":"failed on rule: min","Stock":"failed on rule: min"}}`,
		},
		{
			name: "Error - invalid json",
			mockService: mockProductService{
				product: nil,
				error:   nil,
			},
			productID:    mockID.String(),
			requestBody:  `invalid json`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"Invalid request body"}`,
		},
		{
			name: "Error - product not found",
			mockService: mockProductService{
				product: nil,
				error:   producterrors.ErrProductNotFound,
			},
			productID:    mockID.String(),
			requestBody:  `{"name":"Nonexistent Product","price":100,"stock":10,"version":1}`,
			expectedCode: http.StatusNotFound,
			expectedBody: `{"error":"Product with ID ` + mockID.String() + ` not found"}`,
		},
		{
			name: "Error - service error",
			mockService: mockProductService{
				product: nil,
				error:   errors.New("service unavailable"),
			},
			productID:    mockID.String(),
			requestBody:  `{"name":"Another Product","price":150,"stock":5,"version":1}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to update product with ID ` + mockID.String() + `"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewAPI(&tc.mockService, logger)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+tc.productID, nil)
			req.Body = io.NopCloser(strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("id", tc.productID)
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

func Test_ProductAPI_UpdateStock(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name         string
		mockService  mockProductService
		productID    string
		requestBody  string
		expectedCode int
		expectedBody string
	}{
		{
			name: "Success - stock updated",
			mockService: mockProductService{
				product: &service.ProductDto{ID: mockID.String(), Name: "Product 1", Price: 100, Stock: 30, Version: 1},
				error:   nil,
			},
			productID:    mockID.String(),
			requestBody:  `{"stock":30,"version":1}`,
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"` + mockID.String() + `","name":"Product 1","price":100,"stock":30, "version":1}`,
		},
		{
			name: "Error - validation failed",
			mockService: mockProductService{
				product: nil,
				error:   nil,
			},
			productID:    mockID.String(),
			requestBody:  `{"stock":-10,"version":1}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"validation_errors":{"Stock":"failed on rule: min"}}`,
		},
		{
			name: "Error - service error",
			mockService: mockProductService{
				product: nil,
				error:   errors.New("service unavailable"),
			},
			productID:    mockID.String(),
			requestBody:  `{"stock":25,"version":1}`,
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to update stock for product with ID ` + mockID.String() + `"}`,
		},
		{
			name: "Error - product not found",
			mockService: mockProductService{
				product: nil,
				error:   producterrors.ErrProductNotFound,
			},
			productID:    mockID.String(),
			requestBody:  `{"stock":50,"version":1}`,
			expectedCode: http.StatusNotFound,
			expectedBody: `{"error":"Product with ID ` + mockID.String() + ` not found"}`,
		},
		{
			name: "Error - invalid json",
			mockService: mockProductService{
				product: nil,
				error:   nil,
			},
			productID:    mockID.String(),
			requestBody:  `invalid json`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"Invalid request body"}`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewAPI(&tc.mockService, logger)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/products/"+tc.productID+"/stock", nil)
			req.Body = io.NopCloser(strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")
			req.SetPathValue("id", tc.productID)
			rr := httptest.NewRecorder()

			// when
			api.UpdateStock(rr, req)

			// then
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			assert.Equal(t, tc.expectedCode, rr.Code, "status code should match")
			assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "response body should match")
		})
	}
}

func Test_ProductAPI_DeleteByID(t *testing.T) {
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name         string
		mockService  mockProductService
		productID    string
		expectedCode int
		expectedBody string
		urlParams    string
	}{
		{
			name: "Success - product deleted",
			mockService: mockProductService{
				error: nil,
			},
			productID:    mockID.String(),
			expectedCode: http.StatusNoContent,
			expectedBody: "",
			urlParams:    "?version=1",
		},
		{
			name: "Error - product not found",
			mockService: mockProductService{
				error: producterrors.ErrProductNotFound,
			},
			productID:    mockID.String(),
			expectedCode: http.StatusNotFound,
			expectedBody: `{"error":"Product with ID ` + mockID.String() + ` not found"}`,
			urlParams:    "?version=1",
		},
		{
			name: "Error - service error",
			mockService: mockProductService{
				error: errors.New("service unavailable"),
			},
			productID:    mockID.String(),
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to delete product with ID ` + mockID.String() + `"}`,
			urlParams:    "?version=1",
		},
		{
			name: "Error - version url parameter is required",
			mockService: mockProductService{
				error: nil,
			},
			productID:    mockID.String(),
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"error":"version url parameter is required"}`,
			urlParams:    "", // No version provided
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			api := NewAPI(&tc.mockService, logger)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+tc.productID+tc.urlParams, nil)
			req.SetPathValue("id", tc.productID)
			rr := httptest.NewRecorder()

			// when
			api.DeleteByID(rr, req)

			// then
			assert.Equal(t, tc.expectedCode, rr.Code, "status code should match")
			assert.Equal(t, tc.expectedBody, rr.Body.String(), "response body should match")
		})
	}
}

func Test_ProductAPI_HealthCheck(t *testing.T) {
	// given
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	api := NewAPI(nil, logger) // No service needed for health check
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rr := httptest.NewRecorder()

	// when
	api.HealthCheck(rr, req)

	// then
	assert.Equal(t, http.StatusOK, rr.Code, "status code should be 200 OK")
	assert.Empty(t, rr.Body.String(), "response body should be empty")
}
