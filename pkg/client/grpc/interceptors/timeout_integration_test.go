package interceptors

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// slowProductService is a mock gRPC service that simulates a slow response
type slowProductService struct {
	pb.UnimplementedProductServiceServer
	delay time.Duration
}

func (s *slowProductService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	time.Sleep(s.delay)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	return &pb.GetProductResponse{Products: []*pb.Product{{Id: req.Products[0]}}}, nil
}

// skipIntegrationTests is an environment variable that can be set to skip integration tests
const skipIntegrationTests = "PKG_SKIP_INTEGRATION_TESTS"

// Test_GRPCClient_TimeoutInterceptor tests the gRPC client timeout interceptor
func Test_GRPCClient_TimeoutInterceptor(t *testing.T) {
	if os.Getenv(skipIntegrationTests) == "1" {
		t.Skip("Skipping integration tests based on " + skipIntegrationTests + " env var")
	}
	// given
	const serviceDelay = 200 * time.Millisecond
	const clientTimeout = 100 * time.Millisecond

	// 1. start slow grpc server
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	slowService := &slowProductService{delay: serviceDelay}
	pb.RegisterProductServiceServer(grpcServer, slowService)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	t.Cleanup(func() { grpcServer.Stop() })

	// 2. create gRPC client with shorter timeout
	grpcClient, err := grpc.NewClient(
		lis.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(
			UnaryClientTimeoutInterceptor(clientTimeout),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = grpcClient.Close() })

	productClient := pb.NewProductServiceClient(grpcClient)

	// when
	_, err = productClient.GetProduct(context.Background(), &pb.GetProductRequest{Products: []string{uuid.NewString()}})

	// then
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "Error should be a gRPC status error")
	require.Equal(t, codes.DeadlineExceeded, st.Code(), "Expected DeadlineExceeded error code")
}
