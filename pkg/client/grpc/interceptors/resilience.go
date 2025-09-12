package interceptors

import (
	"context"
	"time"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/sony/gobreaker/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewRetryInterceptor creates a gRPC unary client interceptor with retry logic.
func NewRetryInterceptor(cfg config.RetryConfig) grpc.UnaryClientInterceptor {
	opts := []retry.CallOption{
		// Retry on transient errors.
		retry.WithCodes(codes.Unavailable, codes.ResourceExhausted, codes.Aborted),
		retry.WithMax(cfg.MaxAttempts),
		retry.WithBackoff(retry.BackoffExponential(cfg.InitialBackoff)),
	}
	return retry.UnaryClientInterceptor(opts...)
}

// UnaryCircuitBreakerInterceptor returns a gRPC unary client interceptor that wraps calls in a Circuit Breaker.
// The CircuitBreaker instance should be configured with a custom `IsSuccessful` function
// to distinguish between system failures (which should trip the breaker) and other errors (like NotFound).
func UnaryCircuitBreakerInterceptor[T any](cb *gobreaker.CircuitBreaker[T]) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// The generic type `T` is not used for the result, as the interceptor only passes through the reply.
		// We only care about the error, which will be evaluated by the circuit breaker's `IsSuccessful` function.
		var zero T
		_, err := cb.Execute(func() (T, error) {
			err := invoker(ctx, method, req, reply, cc, opts...)
			return zero, err
		})
		return err
	}
}

func NewCircuitBreaker(cfg config.CircuitBreakerConfig) grpc.UnaryClientInterceptor {
	st := gobreaker.Settings{
		Name:        "product-service-cb",
		MaxRequests: 3,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > cfg.ConsecutiveFailures ||
				(counts.TotalSuccesses+counts.TotalFailures > cfg.ConsecutiveFailures &&
					float64(counts.TotalFailures)/float64(counts.TotalSuccesses+counts.TotalFailures)*100 > float64(cfg.ErrorRatePercent))
		},
		IsSuccessful: func(err error) bool {
			if err == nil {
				return true
			}
			st, ok := status.FromError(err)
			if !ok {
				// Not a gRPC status error, treat as a failure.
				return false
			}
			switch st.Code() {
			case codes.Unavailable, codes.ResourceExhausted, codes.Aborted:
				return false // Report as failure.
			default:
				// For other gRPC errors (e.g., NotFound, InvalidArgument),
				// don't count as a system failure for the circuit breaker.
				return true
			}
		},
	}
	breaker := gobreaker.NewCircuitBreaker[any](st)
	return UnaryCircuitBreakerInterceptor(breaker)
}
