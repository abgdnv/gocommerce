package web

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

const XUserId = "X-User-Id"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from the request header
		userID := r.Header.Get(XUserId)
		if userID == "" {
			http.Error(w, "Unauthorized: Missing X-User-Id header", http.StatusUnauthorized)
			return
		}

		// Create a new context with the user ID
		ctx := context.WithValue(r.Context(), UserIDKey, userID)

		// Pass the new context to the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDInjector creates a middleware that injects request id
func RequestIDInjector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())
		if reqID == "" {
			reqID = uuid.NewString()
		}
		ctx := WithRequestID(r.Context(), reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// StructuredLogger creates a middleware that logs HTTP requests in a structured format.
func StructuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			// Get request ID from context and use it to create a structured logger
			reqID := middleware.GetReqID(r.Context())
			requestLogger := logger.With("request_id", reqID)

			defer func() {
				requestLogger.Info("Request completed",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes_written", ww.BytesWritten(),
					"duration_ms", float64(time.Since(start).Nanoseconds())/1e6,
					"remote_addr", r.RemoteAddr,
					"user_agent", r.UserAgent(),
				)
			}()
			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

// Recoverer is a middleware that recovers from panics and logs them using the provided logger.
func Recoverer(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					logger.Error("Panic recovered",
						"panic", rvr,
						"request_id", middleware.GetReqID(r.Context()),
					)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// GetRequestID retrieves the request ID from the context.
// Returns the request ID and a boolean indicating whether it was found.
func GetRequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey).(string)
	return id, ok
}
