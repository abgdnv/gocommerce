// Package rest provides HTTP handlers for order-related operations.
package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	ordererrors "github.com/abgdnv/gocommerce/order_service/internal/errors"
	"github.com/abgdnv/gocommerce/order_service/internal/service"
	"github.com/abgdnv/gocommerce/pkg/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	service  service.OrderService
	validate *validator.Validate
	logger   *slog.Logger
}

// NewHandler creates a new instance of OrderAPI with the provided service.
func NewHandler(service service.OrderService, logger *slog.Logger) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),

		logger: logger.With("component", "rest"),
	}
}

// RegisterRoutes registers the HTTP routes for the order service.
func (h *Handler) RegisterRoutes(r *chi.Mux) {
	r.Group(func(r chi.Router) {
		r.Use(web.AuthMiddleware)
		r.Route("/api/v1/orders", func(r chi.Router) {
			r.Get("/", h.FindOrdersByUserID)
			r.Post("/", h.Create)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", h.FindByID)
				r.Put("/", h.Update)
			})
		})
	})
	r.Get("/healthz", h.HealthCheck)
}

// FindByID retrieves an order by its ID.
func (h *Handler) FindByID(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	// Parse the order ID from the request URL.
	id, ok := web.ParseID(w, r, mLogger)
	if !ok {
		return
	}

	userID, ok := web.GetUserID(w, r, mLogger)
	if !ok {
		return
	}

	mLogger.DebugContext(r.Context(), "Received request to find order by ID", "ID", id)
	found, err := h.service.FindByID(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, ordererrors.ErrOrderNotFound) {
			mLogger.WarnContext(r.Context(), "Order not found", "ID", id)
			web.RespondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Order with ID %s not found", id))
			return
		} else if errors.Is(err, ordererrors.ErrAccessDenied) {
			mLogger.WarnContext(r.Context(), "Access denied to order", "ID", id, "UserID", userID)
			web.RespondError(w, mLogger, http.StatusForbidden, fmt.Sprintf("Access denied to order with ID %s", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error retrieving order", "ID", id, "error", err)
		web.RespondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve order with ID %s", id))
		return
	}
	mLogger.DebugContext(r.Context(), "Successfully retrieved order", slog.String("ID", found.ID.String()))
	web.RespondJSON(w, mLogger, http.StatusOK, found)

}

// FindOrdersByUserID retrieves a list of all orders.
func (h *Handler) FindOrdersByUserID(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	limit, ok := web.ParseValidateGt(r, w, mLogger, "limit", 0)
	if !ok {
		return
	}
	offset, ok := web.ParseValidateGte(r, w, mLogger, "offset", 0)
	if !ok {
		return
	}
	userID, ok := web.GetUserID(w, r, mLogger)
	if !ok {
		return
	}

	mLogger.DebugContext(r.Context(), "Received request to find all orders", "limit", limit, "offset", offset)
	list, err := h.service.FindOrdersByUserID(r.Context(), userID, offset, limit)
	if err != nil && errors.Is(err, ordererrors.ErrAccessDenied) {
		mLogger.WarnContext(r.Context(), "Access denied to order list", "UserID", userID)
		web.RespondError(w, mLogger, http.StatusForbidden, "Access denied")
		return
	} else if err != nil {
		mLogger.ErrorContext(r.Context(), "Error retrieving order list", "error", err)
		web.RespondError(w, mLogger, http.StatusInternalServerError, "Failed to fetch orders")
		return
	}
	mLogger.DebugContext(r.Context(), "Successfully retrieved order list", "count", len(*list))
	web.RespondJSON(w, mLogger, http.StatusOK, *list)
}

// Create handles the creation of a new order.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	userID, ok := web.GetUserID(w, r, mLogger)
	if !ok {
		return
	}
	var OrderCreateDto service.OrderCreateDto
	if err := json.NewDecoder(r.Body).Decode(&OrderCreateDto); err != nil {
		mLogger.ErrorContext(r.Context(), "Error decoding request body", "error", err)
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}
	// Set the user ID in the order creation DTO.
	OrderCreateDto.UserID = userID

	mLogger.DebugContext(r.Context(), "Received request to create order", "order", OrderCreateDto)
	if err := h.validate.Struct(OrderCreateDto); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			// If the error is a validation error, we can extract field-specific errors.
			errorResponse := make(map[string]string)
			for _, fieldErr := range validationErrors {
				// fieldErr.Tag() returns "required", "max", etc.
				errorResponse[fieldErr.Field()] = "failed on rule: " + fieldErr.Tag()
			}
			mLogger.WarnContext(r.Context(), "Validation errors occurred", "errors", errorResponse)
			web.RespondJSON(w, mLogger, http.StatusBadRequest, map[string]any{"validation_errors": errorResponse})
			return
		}
		mLogger.ErrorContext(r.Context(), "Error validating request body", "error", err)
		// If it's not a validation error, we can return a generic error.
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	newOrder, err := h.service.Create(r.Context(), OrderCreateDto)
	if err != nil && errors.Is(err, ordererrors.ErrInsufficientStock) {
		web.RespondError(w, mLogger, http.StatusBadRequest, err.Error())
		return
	} else if err != nil {
		errStatus, message := web.MapGrpcToHttpStatus(err)
		web.RespondError(w, mLogger, errStatus, message)
		return
	}
	mLogger.InfoContext(r.Context(), "Order created successfully", slog.String("ID", newOrder.ID.String()))
	web.RespondJSON(w, mLogger, http.StatusCreated, newOrder)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	id, ok := web.ParseID(w, r, mLogger)
	if !ok {
		return
	}
	userID, ok := web.GetUserID(w, r, mLogger)
	if !ok {
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to update order", "ID", id)
	var orderUpdateDto service.OrderUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&orderUpdateDto); err != nil {
		mLogger.ErrorContext(r.Context(), "Error decoding request body", "error", err)
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Set the ID in the order update DTO.
	orderUpdateDto.ID = id

	if err := h.validate.Struct(orderUpdateDto); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			errorResponse := make(map[string]string)
			for _, fieldErr := range validationErrors {
				errorResponse[fieldErr.Field()] = "failed on rule: " + fieldErr.Tag()
			}
			mLogger.WarnContext(r.Context(), "Validation errors occurred", "errors", errorResponse)
			web.RespondJSON(w, mLogger, http.StatusBadRequest, map[string]any{"validation_errors": errorResponse})
			return
		}
		mLogger.ErrorContext(r.Context(), "Error validating request body", "error", err)
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := h.service.Update(r.Context(), userID, orderUpdateDto)
	if err != nil {
		if errors.Is(err, ordererrors.ErrOrderNotFound) {
			mLogger.WarnContext(r.Context(), "Order not found for update", "ID", id)
			web.RespondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Order with ID %s not found", id))
			return
		} else if errors.Is(err, ordererrors.ErrOptimisticLock) {
			mLogger.WarnContext(r.Context(), "Optimistic lock error during order update", "ID", id)
			web.RespondError(w, mLogger, http.StatusConflict, fmt.Sprintf("Order with ID %s has been modified by another user", id))
			return
		} else if errors.Is(err, ordererrors.ErrAccessDenied) {
			mLogger.WarnContext(r.Context(), "Access denied to order update", "ID", id, "UserID", userID)
			web.RespondError(w, mLogger, http.StatusForbidden, fmt.Sprintf("Access denied to order with ID %s", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error updating order", "ID", id, "error", err)
		web.RespondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to update order with ID %s", id))
		return
	}
	mLogger.InfoContext(r.Context(), "Order updated successfully", slog.String("ID", updated.ID.String()))
	web.RespondJSON(w, mLogger, http.StatusOK, updated)
}

// HealthCheck is a simple health check endpoint.
func (h *Handler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// loggerWithReqID creates a logger with the request ID from the context.
func (h *Handler) loggerWithReqID(r *http.Request) *slog.Logger {
	reqID := middleware.GetReqID(r.Context())
	return h.logger.With("request_id", reqID)
}
