package handler

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	producterrors "github.com/abgdnv/gocommerce/internal/product/errors"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/stretchr/testify/assert"
)

// mockProductService is a mock implementation of the ProductService interface
type mockProductService struct {
	product  service.ProductDto
	products []service.ProductDto
	error    error
}

// Simulate finding a product by ID
func (m mockProductService) FindByID(_ string) (*service.ProductDto, error) {
	return &m.product, m.error
}

func (m mockProductService) FindAll() (*[]service.ProductDto, error) {
	return &m.products, m.error
}

// Simulate creating a product
func (m mockProductService) Create(_ service.ProductDto) (*service.ProductDto, error) {
	return &m.product, m.error
}

// Simulate deleting a product by ID
func (m mockProductService) DeleteByID(_ string) error {
	return m.error
}

func Test_ProductAPI_FindByID(t *testing.T) {
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
				product: service.ProductDto{ID: "1", Name: "Product 1", Price: 100, Stock: 10},
				error:   nil,
			},
			productID:    "1",
			expectedCode: http.StatusOK,
			expectedBody: `{"id":"1","name":"Product 1","price":100,"stock":10}`,
		},
		{
			name: "Error - product not found",
			mockService: mockProductService{
				product: service.ProductDto{},
				error:   producterrors.ErrProductNotFound,
			},
			productID:    "999",
			expectedCode: http.StatusNotFound,
			expectedBody: `{"error":"Product with ID 999 not found"}`,
		},
		{
			name: "Error - service error",
			mockService: mockProductService{
				product: service.ProductDto{},
				error:   errors.New("service unavailable"),
			},
			productID:    "2",
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to retrieve product with ID 2"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			api := NewAPI(&tc.mockService)
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
		name         string
		mockService  mockProductService
		expectedCode int
		expectedBody string
	}{
		{
			name: "Success - products found",
			mockService: mockProductService{
				products: []service.ProductDto{
					{ID: "1", Name: "Product 1", Price: 100, Stock: 10},
					{ID: "2", Name: "Product 2", Price: 200, Stock: 20},
				},
				error: nil,
			},
			expectedCode: http.StatusOK,
			expectedBody: `[{"id":"1","name":"Product 1","price":100,"stock":10},{"id":"2","name":"Product 2","price":200,"stock":20}]`,
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			api := NewAPI(&tc.mockService)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products", nil)
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
				product: service.ProductDto{ID: "1", Name: "New Product", Price: 150, Stock: 5},
				error:   nil,
			},
			requestBody:  `{"name":"New Product","price":150,"stock":5}`,
			expectedCode: http.StatusCreated,
			expectedBody: `{"id":"1","name":"New Product","price":150,"stock":5}`,
		},
		{
			name: "Error - validation failed",
			mockService: mockProductService{
				product: service.ProductDto{},
				error:   nil,
			},
			requestBody:  `{"name":"","price":-100,"stock":-5}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: `{"validation_errors":{"Name":"failed on rule: required","Price":"failed on rule: min","Stock":"failed on rule: min"}}`,
		},
		{
			name: "Error - service error",
			mockService: mockProductService{
				product: service.ProductDto{},
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
			api := NewAPI(&tc.mockService)
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

func Test_ProductAPI_DeleteByID(t *testing.T) {
	testCases := []struct {
		name         string
		mockService  mockProductService
		productID    string
		expectedCode int
		expectedBody string
	}{
		{
			name: "Success - product deleted",
			mockService: mockProductService{
				error: nil,
			},
			productID:    "1",
			expectedCode: http.StatusNoContent,
			expectedBody: "",
		},
		{
			name: "Error - product not found",
			mockService: mockProductService{
				error: producterrors.ErrProductNotFound,
			},
			productID:    "999",
			expectedCode: http.StatusNotFound,
			expectedBody: `{"error":"Product with ID 999 not found"}`,
		},
		{
			name: "Error - service error",
			mockService: mockProductService{
				error: errors.New("service unavailable"),
			},
			productID:    "2",
			expectedCode: http.StatusInternalServerError,
			expectedBody: `{"error":"Failed to delete product with ID 2"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			api := NewAPI(&tc.mockService)
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/products/"+tc.productID, nil)
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
	api := NewAPI(nil) // No service needed for health check
	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rr := httptest.NewRecorder()

	// when
	api.HealthCheck(rr, req)

	// then
	assert.Equal(t, http.StatusOK, rr.Code, "status code should be 200 OK")
	assert.Empty(t, rr.Body.String(), "response body should be empty")
}
