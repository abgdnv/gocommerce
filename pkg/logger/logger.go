package logger

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
)

// ContextHandler is a wrapper around slog.Handler that adds context information.
type ContextHandler struct {
	slog.Handler
}

// NewContextHandler creates a new ContextHandler.
func NewContextHandler(handler slog.Handler) *ContextHandler {
	return &ContextHandler{
		Handler: handler,
	}
}

// Enabled reports whether the handler records at the given level.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

// Handle processes a log record and adds context information.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
		r.AddAttrs(slog.String("trace_id", span.SpanContext().TraceID().String()))
	}
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		r.AddAttrs(slog.String("request_id", reqID))
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs returns a new ContextHandler with the given attributes added.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{
		Handler: h.Handler.WithAttrs(attrs),
	}
}

// WithGroup returns a new ContextHandler with the given group added.
func (h *ContextHandler) WithGroup(group string) slog.Handler {
	return &ContextHandler{
		Handler: h.Handler.WithGroup(group),
	}
}
