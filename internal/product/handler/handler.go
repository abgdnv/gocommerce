// Package handler provides HTTP handlers for product-related operations.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	producterrors "github.com/abgdnv/gocommerce/internal/product/errors"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/abgdnv/gocommerce/pkg/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Handler struct {
	service  service.ProductService
	validate *validator.Validate
	logger   *slog.Logger
}

// NewHandler creates a new instance of ProductAPI with the provided service.
func NewHandler(service service.ProductService, logger *slog.Logger) *Handler {
	return &Handler{
		service:  service,
		validate: validator.New(),
		logger:   logger.With("component", "handler"),
	}
}

// RegisterRoutes registers the HTTP routes for the product service.
func (h *Handler) RegisterRoutes(r *chi.Mux) {
	r.Route("/api/v1/products", func(r chi.Router) {
		r.Get("/", h.FindAll)
		r.Post("/", h.Create)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", h.FindByID)
			r.Delete("/", h.DeleteByID)
			r.Put("/", h.Update)
			r.Put("/stock", h.UpdateStock)
		})
	})

	r.Get("/healthz", h.HealthCheck)
}

// FindByID retrieves a product by its ID.
func (h *Handler) FindByID(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	id, ok := parseID(w, r, mLogger)
	if !ok {
		return
	}

	mLogger.DebugContext(r.Context(), "Received request to find product by ID", "ID", id)
	found, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			mLogger.WarnContext(r.Context(), "Product not found", "ID", id)
			respondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error retrieving product", "ID", id, "error", err)
		respondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve product with ID %s", id))
		return
	}
	mLogger.DebugContext(r.Context(), "Successfully retrieved product", "ID", found.ID, "Name", found.Name)
	respondJSON(w, mLogger, http.StatusOK, found)

}

// FindAll retrieves a list of all products.
func (h *Handler) FindAll(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	limit, ok := parseValidate(r, w, mLogger, "limit", gt(0))
	if !ok {
		return
	}
	offset, ok := parseValidate(r, w, mLogger, "offset", gte(0))
	if !ok {
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to find all products", "limit", limit, "offset", offset)
	list, err := h.service.FindAll(r.Context(), offset, limit)
	if err != nil {
		mLogger.ErrorContext(r.Context(), "Error retrieving product list", "error", err)
		respondError(w, mLogger, http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	mLogger.DebugContext(r.Context(), "Successfully retrieved product list", "count", len(*list))
	respondJSON(w, mLogger, http.StatusOK, *list)
}

// Create handles the creation of a new product.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	var productCreateDto service.ProductCreateDto
	if err := json.NewDecoder(r.Body).Decode(&productCreateDto); err != nil {
		mLogger.ErrorContext(r.Context(), "Error decoding request body", "error", err)
		respondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to create product", "product", productCreateDto)
	if err := h.validate.Struct(productCreateDto); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			// If the error is a validation error, we can extract field-specific errors.
			errorResponse := make(map[string]string)
			for _, fieldErr := range validationErrors {
				// fieldErr.Tag() returns "required", "max", etc.
				errorResponse[fieldErr.Field()] = "failed on rule: " + fieldErr.Tag()
			}
			mLogger.WarnContext(r.Context(), "Validation errors occurred", "errors", errorResponse)
			respondJSON(w, mLogger, http.StatusBadRequest, map[string]any{"validation_errors": errorResponse})
			return
		}
		mLogger.ErrorContext(r.Context(), "Error validating request body", "error", err)
		// If it's not a validation error, we can return a generic error.
		respondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	newProduct, err := h.service.Create(r.Context(), productCreateDto)
	if err != nil {
		mLogger.ErrorContext(r.Context(), "Error creating product", "error", err)
		respondError(w, mLogger, http.StatusInternalServerError, "Failed to create product")
		return
	}
	mLogger.InfoContext(r.Context(), "Product created successfully", "ID", newProduct.ID, "Name", newProduct.Name)
	respondJSON(w, mLogger, http.StatusCreated, newProduct)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	id, ok := parseID(w, r, mLogger)
	if !ok {
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to update product", "ID", id)
	var productDTO service.ProductDto
	if err := json.NewDecoder(r.Body).Decode(&productDTO); err != nil {
		mLogger.ErrorContext(r.Context(), "Error decoding request body", "error", err)
		respondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validate.Struct(productDTO); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			errorResponse := make(map[string]string)
			for _, fieldErr := range validationErrors {
				errorResponse[fieldErr.Field()] = "failed on rule: " + fieldErr.Tag()
			}
			mLogger.WarnContext(r.Context(), "Validation errors occurred", "errors", errorResponse)
			respondJSON(w, mLogger, http.StatusBadRequest, map[string]any{"validation_errors": errorResponse})
			return
		}
		mLogger.ErrorContext(r.Context(), "Error validating request body", "error", err)
		respondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	productDTO.ID = id.String()

	updated, err := h.service.Update(r.Context(), productDTO)
	if err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			mLogger.WarnContext(r.Context(), "Product not found for update", "ID", id)
			respondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error updating product", "ID", id, "error", err)
		respondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to update product with ID %s", id))
		return
	}
	mLogger.InfoContext(r.Context(), "Product updated successfully", "ID", updated.ID, "Name", updated.Name)
	respondJSON(w, mLogger, http.StatusOK, updated)
}

func (h *Handler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	id, ok := parseID(w, r, mLogger)
	if !ok {
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to update stock for product", "ID", id)
	var stockUpdateDTO service.StockUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&stockUpdateDTO); err != nil {
		mLogger.ErrorContext(r.Context(), "Error decoding request body", "error", err)
		respondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validate.Struct(stockUpdateDTO); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			errorResponse := make(map[string]string)
			for _, fieldErr := range validationErrors {
				errorResponse[fieldErr.Field()] = "failed on rule: " + fieldErr.Tag()
			}
			mLogger.WarnContext(r.Context(), "Validation errors occurred", "errors", errorResponse)
			respondJSON(w, mLogger, http.StatusBadRequest, map[string]any{"validation_errors": errorResponse})
			return
		}
		mLogger.ErrorContext(r.Context(), "Error validating request body", "error", err)
		respondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := h.service.UpdateStock(r.Context(), id, stockUpdateDTO.Stock, stockUpdateDTO.Version)
	if err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			mLogger.WarnContext(r.Context(), "Product not found for stock update", "ID", id)
			respondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error updating stock for product", "ID", id, "error", err)
		respondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to update stock for product with ID %s", id))
		return
	}
	mLogger.InfoContext(r.Context(), "Stock updated successfully for product", "ID", updated.ID, "NewStock", updated.Stock)
	respondJSON(w, mLogger, http.StatusOK, updated)
}

// DeleteByID deletes a product by its ID.
func (h *Handler) DeleteByID(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	id, ok := parseID(w, r, mLogger)
	if !ok {
		return
	}
	version, ok := parseValidate(r, w, mLogger, "version", gte(1))
	if !ok {
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to delete product", "ID", id, "Version", version)
	if err := h.service.DeleteByID(r.Context(), id, version); err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			mLogger.WarnContext(r.Context(), "Product not found for deletion", "ID", id)
			respondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error deleting product", "ID", id, "error", err)
		respondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to delete product with ID %s", id))
		return

	}
	mLogger.InfoContext(r.Context(), "Product deleted successfully", "ID", id)
	w.WriteHeader(http.StatusNoContent)
}

// HealthCheck is a simple health check endpoint.
func (h *Handler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func respondJSON(w http.ResponseWriter, logger *slog.Logger, status int, payload any) {
	// Handle nil payload
	if payload == nil {
		w.WriteHeader(status)
		return
	}

	response, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Error encoding response to JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(response)
}

func respondError(w http.ResponseWriter, logger *slog.Logger, status int, message string) {
	respondJSON(w, logger, status, map[string]string{"error": message})
}

// parseID extracts and validates the product ID from the request path. Returns the ID and a boolean indicating success.
func parseID(w http.ResponseWriter, r *http.Request, logger *slog.Logger) (uuid.UUID, bool) {
	pathValueID := r.PathValue("id")
	id, err := uuid.Parse(pathValueID)
	if err != nil {
		respondError(w, logger, http.StatusBadRequest, fmt.Sprintf("Invalid product ID: %s", pathValueID))
		return uuid.UUID{}, false
	}
	return id, true
}

func parseValidate(r *http.Request, w http.ResponseWriter, logger *slog.Logger, key string, pValidator ParamValidator) (int32, bool) {
	value := r.URL.Query().Get(key)
	if value == "" {
		respondError(w, logger, http.StatusBadRequest, fmt.Sprintf("%s url parameter is required", key))
		return 0, false // Return false if the parameter is not present
	}
	intValue, err := strconv.ParseInt(value, 10, 32)
	if err != nil || !pValidator(intValue) {
		respondError(w, logger, http.StatusBadRequest, fmt.Sprintf("Invalid %s number: %s", key, value))
		return 0, false
	}
	return int32(intValue), true
}

// loggerWithReqID creates a logger with the request ID from the context.
func (h *Handler) loggerWithReqID(r *http.Request) *slog.Logger {
	reqID, found := web.GetRequestID(r.Context())
	if !found {
		reqID = "unknown"
	}
	return h.logger.With("request_id", reqID)
}

// ParamValidator is a function type that validates a parameter.
type ParamValidator func(valueToTest int64) bool

func newComparisonValidator(valueInClosure int64, compareFn func(argValue, closedValue int64) bool) ParamValidator {
	return func(argValue int64) bool {
		return compareFn(argValue, valueInClosure)
	}
}

// gte returns a ParamValidator that checks if the argument is greater than or equal to the value captured in the closure.
func gte(valToCompareAgainst int64) ParamValidator {
	return newComparisonValidator(valToCompareAgainst, func(argValue, closedValue int64) bool {
		return argValue >= closedValue
	})
}

// gt returns a ParamValidator that checks if the argument is greater than the value captured in the closure.
func gt(valToCompareAgainst int64) ParamValidator {
	return newComparisonValidator(valToCompareAgainst, func(argValue, closedValue int64) bool {
		return argValue > closedValue
	})
}
