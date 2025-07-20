package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// UnaryServerTimeoutInterceptor returns a unary server interceptor that applies a timeout to the context of the request.
func UnaryClientTimeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return invoker(callCtx, method, req, reply, cc, opts...)
	}
}
