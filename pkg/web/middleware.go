package web

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

// StructuredLogger creates a middleware that logs HTTP requests in a structured format.
func StructuredLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				logger.InfoContext(r.Context(), "Request completed",
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

// TelemetryEnricher — middleware to enrich OTel spans with additional common tags.
func TelemetryEnricher(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		if !span.IsRecording() {
			next.ServeHTTP(w, r)
			return
		}
		var attrs []attribute.KeyValue
		if routePattern := chi.RouteContext(r.Context()).RoutePattern(); routePattern != "" {
			attrs = append(attrs, attribute.String("http.route", routePattern))
		}
		if reqID := middleware.GetReqID(r.Context()); reqID != "" {
			attrs = append(attrs, attribute.String("http.request_id", reqID))
		}
		span.SetAttributes(attrs...)

		next.ServeHTTP(w, r)
	})
}
