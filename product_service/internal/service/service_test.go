package service

import (
	"context"
	"errors"
	"testing"

	"github.com/abgdnv/gocommerce/product_service/internal/store/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProductStore is a mock implementation of the ProductStore interface
type mockProductStore struct {
	products []db.Product
	product  db.Product
	error    error
}

// Simulate finding a product by ID
func (m *mockProductStore) FindByID(_ context.Context, _ uuid.UUID) (*db.Product, error) {
	return &m.product, m.error
}

// Simulate finding all products
func (m *mockProductStore) FindAll(_ context.Context, _, _ int32) (*[]db.Product, error) {
	return &m.products, m.error
}

// Simulate creating a product
func (m *mockProductStore) Create(_ context.Context, _ string, _ int64, _ int32) (*db.Product, error) {
	return &m.product, m.error
}

// Simulate updating a product
func (m *mockProductStore) Update(_ context.Context, _ uuid.UUID, _ string, _ int64, _ int32, _ int32) (*db.Product, error) {
	return &m.product, m.error
}

// Simulate updating stock for a product
func (m *mockProductStore) UpdateStock(_ context.Context, _ uuid.UUID, _ int32, _ int32) (*db.Product, error) {
	return &m.product, m.error
}

// Simulate deleting a product by ID
func (m *mockProductStore) DeleteByID(_ context.Context, _ uuid.UUID, _ int32) error {
	return m.error
}

func Test_ProductService_FindByID(t *testing.T) {
	ErrProductNotFound := errors.New("product not found")
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name        string
		mockStore   *mockProductStore
		productID   uuid.UUID
		expected    *ProductDto
		expectError error
	}{
		{
			name: "Success - product found",
			mockStore: &mockProductStore{
				product: db.Product{ID: mockID, Name: "Toy"},
				error:   nil,
			},
			productID:   mockID,
			expected:    &ProductDto{ID: mockID.String(), Name: "Toy"},
			expectError: nil,
		},
		{
			name: "Error - product not found",
			mockStore: &mockProductStore{
				error: ErrProductNotFound,
			},
			productID:   mockID,
			expected:    nil,
			expectError: ErrProductNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore)
			// when
			found, err := service.FindByID(context.Background(), tc.productID)
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
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
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
				products: []db.Product{{ID: mockID, Name: "Toy"}},
				error:    nil,
			},
			expectedList: []ProductDto{{ID: mockID.String(), Name: "Toy"}},
			expectError:  nil,
		},
		{
			name: "Success - no products",
			mockStore: &mockProductStore{
				products: []db.Product{},
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
			found, err := service.FindAll(context.Background(), 0, 10)
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
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name        string
		mockStore   *mockProductStore
		product     ProductCreateDto
		expected    *ProductDto
		expectError error
	}{
		{
			name: "Success - product created",
			mockStore: &mockProductStore{
				product: db.Product{ID: mockID, Name: "Toy", Price: 100, StockQuantity: 10},
				error:   nil,
			},
			product:     ProductCreateDto{Name: "Toy", Price: 100, Stock: 10},
			expected:    &ProductDto{ID: mockID.String(), Name: "Toy", Price: 100, Stock: 10},
			expectError: nil,
		},
		{
			name: "Error - store error",
			mockStore: &mockProductStore{
				error: ErrStoreError,
			},
			product:     ProductCreateDto{Name: "Toy", Price: 100, Stock: 10},
			expected:    nil,
			expectError: ErrStoreError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore)
			// when
			created, err := service.Create(context.Background(), tc.product)
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

func Test_ProductService_Update(t *testing.T) {
	ErrProductNotFound := errors.New("product not found")
	ErrStoreError := errors.New("store error")
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name        string
		mockStore   *mockProductStore
		product     ProductDto
		expected    *ProductDto
		expectError error
	}{
		{
			name: "Success - product updated",
			mockStore: &mockProductStore{
				product: db.Product{ID: mockID, Name: "Updated Toy", Price: 150, StockQuantity: 20, Version: 2},
				error:   nil,
			},
			product:     ProductDto{ID: mockID.String(), Name: "Updated Toy", Price: 150, Stock: 20, Version: 2},
			expected:    &ProductDto{ID: mockID.String(), Name: "Updated Toy", Price: 150, Stock: 20, Version: 2},
			expectError: nil,
		},
		{
			name: "Error - product not found",
			mockStore: &mockProductStore{
				error: ErrProductNotFound,
			},
			product:     ProductDto{ID: mockID.String(), Name: "Updated Toy", Price: 150, Stock: 20, Version: 2},
			expected:    nil,
			expectError: ErrProductNotFound,
		},
		{
			name: "Error - store error",
			mockStore: &mockProductStore{
				error: ErrStoreError,
			},
			product:     ProductDto{ID: mockID.String(), Name: "Updated Toy", Price: 150, Stock: 20, Version: 2},
			expected:    nil,
			expectError: ErrStoreError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore)
			// when
			updated, err := service.Update(context.Background(), tc.product)
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

func Test_ProductService_UpdateStock(t *testing.T) {
	ErrProductNotFound := errors.New("product not found")
	ErrStoreError := errors.New("store error")
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name        string
		mockStore   *mockProductStore
		productID   uuid.UUID
		quantity    int32
		version     int32
		expected    *ProductDto
		expectError error
	}{
		{
			name: "Success - stock updated",
			mockStore: &mockProductStore{
				product: db.Product{ID: mockID, Name: "Toy", StockQuantity: 15, Version: 2},
				error:   nil,
			},
			productID:   mockID,
			quantity:    15,
			version:     1,
			expected:    &ProductDto{ID: mockID.String(), Name: "Toy", Stock: 15, Version: 2},
			expectError: nil,
		},
		{
			name: "Error - product not found",
			mockStore: &mockProductStore{
				error: ErrProductNotFound,
			},
			productID:   mockID,
			quantity:    15,
			version:     1,
			expected:    nil,
			expectError: ErrProductNotFound,
		},
		{
			name: "Error - store error",
			mockStore: &mockProductStore{
				error: ErrStoreError,
			},
			productID:   mockID,
			quantity:    15,
			version:     1,
			expected:    nil,
			expectError: ErrStoreError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore)
			// when
			updated, err := service.UpdateStock(context.Background(), tc.productID, tc.quantity, tc.version)
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

func Test_ProductService_DeleteByID(t *testing.T) {
	ErrProductNotFound := errors.New("product not found")
	ErrStoreError := errors.New("store error")
	mockID, _ := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	testCases := []struct {
		name        string
		mockStore   *mockProductStore
		productID   uuid.UUID
		expectError error
	}{
		{
			name: "Success - product deleted",
			mockStore: &mockProductStore{
				error: nil,
			},
			productID:   mockID,
			expectError: nil,
		},
		{
			name: "Error - product not found",
			mockStore: &mockProductStore{
				error: ErrProductNotFound,
			},
			productID:   mockID,
			expectError: ErrProductNotFound,
		},
		{
			name: "Error - store error",
			mockStore: &mockProductStore{
				error: ErrStoreError,
			},
			productID:   mockID,
			expectError: ErrStoreError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			service := NewService(tc.mockStore)
			// when
			err := service.DeleteByID(context.Background(), tc.productID, 1)
			// then
			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)
				return
			}
			require.NoError(t, err)
		})
	}
}
