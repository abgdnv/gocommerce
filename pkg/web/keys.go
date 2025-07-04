package web

import "context"

type requestIDKey struct{}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// GetRequestID retrieves the request ID from the context.
// Returns the request ID and a boolean indicating whether it was found.
func GetRequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey{}).(string)
	return id, ok
}
