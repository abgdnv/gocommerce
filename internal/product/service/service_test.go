package service

import (
	"errors"
	"testing"

	"github.com/abgdnv/gocommerce/internal/product/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProductStore is a mock implementation of the ProductStore interface
type mockProductStore struct {
	products []store.Product
	product  store.Product
	error    error
}

// Simulate finding a product by ID
func (m *mockProductStore) FindByID(_ string) (*store.Product, error) {
	return &m.product, m.error
}

// Simulate finding all products
func (m *mockProductStore) FindAll() (*[]store.Product, error) {
	return &m.products, m.error
}

// Simulate creating a product
func (m *mockProductStore) Create(_ string, _ int64, _ int32) (*store.Product, error) {
	return &m.product, m.error
}

// Simulate deleting a product by ID
func (m *mockProductStore) DeleteByID(_ string) error {
	return m.error
}

func Test_ProductService_FindByID(t *testing.T) {
	ErrProductNotFound := errors.New("product not found")
	testCases := []struct {
		name        string
		mockStore   *mockProductStore
		productID   string
		expected    *ProductDto
		expectError error
	}{
		{
			name: "Success - product found",
			mockStore: &mockProductStore{
				product: store.Product{ID: "1", Name: "Toy"},
				error:   nil,
			},
			productID:   "1",
			expected:    &ProductDto{ID: "1", Name: "Toy"},
			expectError: nil,
		},
		{
			name: "Error - product not found",
			mockStore: &mockProductStore{
				error: ErrProductNotFound,
			},
			productID:   "2",
			expected:    nil,
			expectError: ErrProductNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore)
			// when
			found, err := service.FindByID(tc.productID)
			// then
			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)
				assert.Nil(t, found)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, found)
		})
	}
}

func Test_ProductService_FindAll(t *testing.T) {
	ErrStoreError := errors.New("store error")
	testCases := []struct {
		name         string
		mockStore    *mockProductStore
		expectedList []ProductDto
		expected     []ProductDto
		expectError  error
	}{
		{
			name: "Success - products found",
			mockStore: &mockProductStore{
				products: []store.Product{{ID: "1", Name: "Toy"}},
				error:    nil,
			},
			expectedList: []ProductDto{{ID: "1", Name: "Toy"}},
			expectError:  nil,
		},
		{
			name: "Success - no products",
			mockStore: &mockProductStore{
				products: []store.Product{},
				error:    nil,
			},
			expectedList: []ProductDto{},
			expectError:  nil,
		},
		{
			name: "Error - store error",
			mockStore: &mockProductStore{
				error: ErrStoreError,
			},
			expectedList: nil,
			expectError:  ErrStoreError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore)
			// when
			found, err := service.FindAll()
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

func Test_ProductService_Create(t *testing.T) {
	ErrStoreError := errors.New("store error")
	testCases := []struct {
		name        string
		mockStore   *mockProductStore
		product     ProductDto
		expected    *ProductDto
		expectError error
	}{
		{
			name: "Success - product created",
			mockStore: &mockProductStore{
				product: store.Product{ID: "1", Name: "Toy", Price: 100, Stock: 10},
				error:   nil,
			},
			product:     ProductDto{Name: "Toy", Price: 100, Stock: 10},
			expected:    &ProductDto{ID: "1", Name: "Toy", Price: 100, Stock: 10},
			expectError: nil,
		},
		{
			name: "Error - store error",
			mockStore: &mockProductStore{
				error: ErrStoreError,
			},
			product:     ProductDto{Name: "Toy", Price: 100, Stock: 10},
			expected:    nil,
			expectError: ErrStoreError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore)
			// when
			created, err := service.Create(tc.product)
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

func Test_ProductService_DeleteByID(t *testing.T) {
	ErrProductNotFound := errors.New("product not found")
	ErrStoreError := errors.New("store error")
	testCases := []struct {
		name        string
		mockStore   *mockProductStore
		productID   string
		expectError error
	}{
		{
			name: "Success - product deleted",
			mockStore: &mockProductStore{
				error: nil,
			},
			productID:   "1",
			expectError: nil,
		},
		{
			name: "Error - product not found",
			mockStore: &mockProductStore{
				error: ErrProductNotFound,
			},
			productID:   "2",
			expectError: ErrProductNotFound,
		},
		{
			name: "Error - store error",
			mockStore: &mockProductStore{
				error: ErrStoreError,
			},
			productID:   "3",
			expectError: ErrStoreError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore)
			// when
			err := service.DeleteByID(tc.productID)
			// then
			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)
				return
			}
			require.NoError(t, err)
		})
	}
}
