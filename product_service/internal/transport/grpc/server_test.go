package grpc

import (
	"context"
	"errors"
	"testing"

	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	perrors "github.com/abgdnv/gocommerce/product_service/internal/errors"
	"github.com/abgdnv/gocommerce/product_service/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockProductService struct {
	mock.Mock
}

func (m *MockProductService) FindByID(ctx context.Context, id uuid.UUID) (*service.ProductDto, error) {
	args := m.Called(ctx, id)

	var product *service.ProductDto
	if args.Get(0) != nil {
		product = args.Get(0).(*service.ProductDto)
	}

	return product, args.Error(1)
}

func TestProductService_GetProduct(t *testing.T) {
	ctx := context.Background()
	productID := uuid.New()

	testCases := []struct {
		name         string
		mockProduct  *service.ProductDto
		mockError    error
		expectedCode codes.Code
	}{
		{
			name:         "success",
			mockProduct:  &service.ProductDto{ID: productID.String(), Name: "Test Product", Price: 10.0, Stock: 5},
			expectedCode: codes.OK,
		},
		{
			name:         "not found",
			mockProduct:  nil,
			mockError:    perrors.ErrProductNotFound,
			expectedCode: codes.NotFound,
		},
		{
			name:         "internal error",
			mockProduct:  nil,
			mockError:    errors.New("internal error"),
			expectedCode: codes.Internal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			mockSvc := new(MockProductService)
			server := NewServer(mockSvc)

			mockSvc.On("FindByID", mock.Anything, productID).Return(tc.mockProduct, tc.mockError)

			// when
			req := &pb.GetProductRequest{Id: productID.String()}
			res, err := server.GetProduct(ctx, req)

			// then
			if tc.expectedCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tc.mockProduct.ID, res.Product.Id)
				require.Equal(t, tc.mockProduct.Name, res.Product.Name)
				require.Equal(t, tc.mockProduct.Price, res.Product.Price)
				require.Equal(t, tc.mockProduct.Stock, res.Product.StockQuantity)
			} else {
				require.Nil(t, res)
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tc.expectedCode, st.Code())
			}
			mockSvc.AssertExpectations(t)
		})
	}

	t.Run("invalid id format", func(t *testing.T) {
		// given
		mockSvc := new(MockProductService)
		server := NewServer(mockSvc)

		req := &pb.GetProductRequest{Id: "this-is-not-a-uuid"}

		// when
		_, err := server.GetProduct(ctx, req)

		// then
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
		mockSvc.AssertNotCalled(t, "FindByID", mock.Anything, mock.Anything)
	})

}
