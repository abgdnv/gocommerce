package web

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/abgdnv/gocommerce/internal/platform/contextkeys"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

// RequestIDInjector creates a middleware that injects request id
func RequestIDInjector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(r.Context())
		if reqID == "" {
			reqID = uuid.NewString()
		}
		ctx := contextkeys.WithRequestID(r.Context(), reqID)
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
