package interceptors

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// mockService is a mock implementation of the ProductServiceServer for testing purposes.
// Not thread-safe, should be used in sequential tests only.
type mockService struct {
	pb.UnimplementedProductServiceServer

	callCount int32
	// responses - a queue of gRPC codes to return for each call.
	responses []codes.Code
}

// GetProduct simulates a gRPC call and returns a pre-configured response.
func (s *mockService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	s.callCount++

	if len(s.responses) > 0 {
		code := s.responses[0]
		s.responses = s.responses[1:] // Dequeue the first response code
		if code != codes.OK {
			return nil, status.Error(code, "mock error")
		}
	}

	return &pb.GetProductResponse{}, nil
}

// setResponses configures the sequence of responses for the test server.
func (s *mockService) setResponses(responses ...codes.Code) {
	s.responses = responses
	s.callCount = 0
}

// getCallCount returns the number of times the server has been called.
func (s *mockService) getCallCount() int32 {
	return s.callCount
}

// setupTestEnvironment creates a test gRPC server, a client with interceptors, and a cleanup function.
func setupTestEnvironment(t *testing.T) (client pb.ProductServiceClient, service *mockService, cleanup func()) {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	service = &mockService{}

	grpcServer := grpc.NewServer()
	pb.RegisterProductServiceServer(grpcServer, service)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	retryCfg := config.RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 100 * time.Millisecond,
	}
	circuitBreakerCfg := config.CircuitBreakerConfig{
		ConsecutiveFailures: 5,
		ErrorRatePercent:    60,
		OpenTimeout:         5 * time.Second,
	}

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(
			NewRetryInterceptor(retryCfg),
			NewCircuitBreaker(circuitBreakerCfg),
		),
	)
	require.NoError(t, err)

	client = pb.NewProductServiceClient(conn)

	cleanup = func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = lis.Close()
	}

	return client, service, cleanup
}

func TestInterceptors_HappyPath(t *testing.T) {
	client, service, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// given
	service.setResponses(codes.OK)

	// when
	_, err := client.GetProduct(context.Background(), &pb.GetProductRequest{})

	// then
	require.NoError(t, err)
	require.Equal(t, int32(1), service.getCallCount(), "Server should be called exactly once")
}

func TestInterceptors_RetryOnTransientError(t *testing.T) {
	client, service, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// given
	service.setResponses(codes.Unavailable, codes.Unavailable, codes.OK)

	// when
	_, err := client.GetProduct(context.Background(), &pb.GetProductRequest{})

	// then
	require.NoError(t, err)
	require.Equal(t, int32(3), service.getCallCount(), "Server should be called exactly 3 times due to retries")
}

func TestInterceptors_NoRetryOnDataError(t *testing.T) {
	client, service, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// given
	service.setResponses(codes.InvalidArgument)

	// when
	_, err := client.GetProduct(context.Background(), &pb.GetProductRequest{})

	// then
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Equal(t, int32(1), service.getCallCount(), "Server should be called exactly once, no retries on data error")
}

func TestInterceptors_CircuitBreakerOpens(t *testing.T) {
	client, service, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// given
	// In order for the circuit breaker to open (ConsecutiveFailures > 5),
	// we need 2 calls, each of which will trigger 3 attempts (1 initial + 2 retries).
	// 2 * 3 = 6 failed attempts.
	service.setResponses(
		codes.Unavailable, codes.Unavailable, codes.Unavailable, // first call
		codes.Unavailable, codes.Unavailable, codes.Unavailable, // second call
	)

	// when: call 2 times, it should lead to circuit breaker opening.
	_, err := client.GetProduct(context.Background(), &pb.GetProductRequest{})
	require.Error(t, err, "First call should fail")

	_, err = client.GetProduct(context.Background(), &pb.GetProductRequest{})
	require.Error(t, err, "Second call should fail")

	// check if server was called 6 times (2 calls * 3 attempts).
	require.Equal(t, int32(6), service.getCallCount(), "Server should be called 6 times")

	// then: 3rd call should be immediately blocked.
	_, err = client.GetProduct(context.Background(), &pb.GetProductRequest{})
	require.Error(t, err, "Third call should be blocked by circuit breaker")
	require.Contains(t, gobreaker.ErrOpenState.Error(), err.Error())

	// check if server was called 6 times (2 calls * 3 attempts).
	require.Equal(t, int32(6), service.getCallCount(), "Server call count should not change, circuit breaker should block the call")
}

func TestInterceptors_CircuitBreakerIgnoresDataError(t *testing.T) {
	client, service, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// given
	responses := make([]codes.Code, 10)
	for i := range responses {
		responses[i] = codes.InvalidArgument
	}
	service.setResponses(responses...)

	// when
	for i := 0; i < 10; i++ {
		_, err := client.GetProduct(context.Background(), &pb.GetProductRequest{})
		// then
		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.InvalidArgument, st.Code())
	}

	// then
	require.Equal(t, int32(10), service.getCallCount(), "Server should be called exactly 10 times, circuit breaker should not trigger on data errors")
}
