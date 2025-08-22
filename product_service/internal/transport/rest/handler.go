// Package rest provides HTTP handlers for product-related operations.
package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/abgdnv/gocommerce/pkg/web"
	producterrors "github.com/abgdnv/gocommerce/product_service/internal/errors"
	"github.com/abgdnv/gocommerce/product_service/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
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
		logger:   logger.With("component", "rest"),
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
	id, ok := web.ParseID(w, r, mLogger)
	if !ok {
		return
	}

	mLogger.DebugContext(r.Context(), "Received request to find product by ID", "ID", id)
	found, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			mLogger.WarnContext(r.Context(), "Product not found", "ID", id)
			web.RespondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error retrieving product", "ID", id, "error", err)
		web.RespondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve product with ID %s", id))
		return
	}
	mLogger.DebugContext(r.Context(), "Successfully retrieved product", "ID", found.ID, "Name", found.Name)
	web.RespondJSON(w, mLogger, http.StatusOK, found)

}

// FindAll retrieves a list of all products.
func (h *Handler) FindAll(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	limit, ok := web.ParseValidateGt(r, w, mLogger, "limit", 0)
	if !ok {
		return
	}
	offset, ok := web.ParseValidateGte(r, w, mLogger, "offset", 0)
	if !ok {
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to find all products", "limit", limit, "offset", offset)
	list, err := h.service.FindAll(r.Context(), offset, limit)
	if err != nil {
		mLogger.ErrorContext(r.Context(), "Error retrieving product list", "error", err)
		web.RespondError(w, mLogger, http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	mLogger.DebugContext(r.Context(), "Successfully retrieved product list", "count", len(list))
	web.RespondJSON(w, mLogger, http.StatusOK, list)
}

// Create handles the creation of a new product.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	var productCreateDto service.ProductCreateDto
	if err := json.NewDecoder(r.Body).Decode(&productCreateDto); err != nil {
		mLogger.ErrorContext(r.Context(), "Error decoding request body", "error", err)
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
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
			web.RespondJSON(w, mLogger, http.StatusBadRequest, map[string]any{"validation_errors": errorResponse})
			return
		}
		mLogger.ErrorContext(r.Context(), "Error validating request body", "error", err)
		// If it's not a validation error, we can return a generic error.
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	newProduct, err := h.service.Create(r.Context(), productCreateDto)
	if err != nil {
		mLogger.ErrorContext(r.Context(), "Error creating product", "error", err)
		web.RespondError(w, mLogger, http.StatusInternalServerError, "Failed to create product")
		return
	}
	mLogger.InfoContext(r.Context(), "Product created successfully", "ID", newProduct.ID, "Name", newProduct.Name)
	web.RespondJSON(w, mLogger, http.StatusCreated, newProduct)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	id, ok := web.ParseID(w, r, mLogger)
	if !ok {
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to update product", "ID", id)
	var productDTO service.ProductDto
	if err := json.NewDecoder(r.Body).Decode(&productDTO); err != nil {
		mLogger.ErrorContext(r.Context(), "Error decoding request body", "error", err)
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
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
			web.RespondJSON(w, mLogger, http.StatusBadRequest, map[string]any{"validation_errors": errorResponse})
			return
		}
		mLogger.ErrorContext(r.Context(), "Error validating request body", "error", err)
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	productDTO.ID = id.String()

	updated, err := h.service.Update(r.Context(), productDTO)
	if err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			mLogger.WarnContext(r.Context(), "Product not found for update", "ID", id)
			web.RespondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error updating product", "ID", id, "error", err)
		web.RespondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to update product with ID %s", id))
		return
	}
	mLogger.InfoContext(r.Context(), "Product updated successfully", "ID", updated.ID, "Name", updated.Name)
	web.RespondJSON(w, mLogger, http.StatusOK, updated)
}

func (h *Handler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	id, ok := web.ParseID(w, r, mLogger)
	if !ok {
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to update stock for product", "ID", id)
	var stockUpdateDTO service.StockUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&stockUpdateDTO); err != nil {
		mLogger.ErrorContext(r.Context(), "Error decoding request body", "error", err)
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
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
			web.RespondJSON(w, mLogger, http.StatusBadRequest, map[string]any{"validation_errors": errorResponse})
			return
		}
		mLogger.ErrorContext(r.Context(), "Error validating request body", "error", err)
		web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := h.service.UpdateStock(r.Context(), id, stockUpdateDTO.Stock, stockUpdateDTO.Version)
	if err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			mLogger.WarnContext(r.Context(), "Product not found for stock update", "ID", id)
			web.RespondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error updating stock for product", "ID", id, "error", err)
		web.RespondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to update stock for product with ID %s", id))
		return
	}
	mLogger.InfoContext(r.Context(), "Stock updated successfully for product", "ID", updated.ID, "NewStock", updated.Stock)
	web.RespondJSON(w, mLogger, http.StatusOK, updated)
}

// DeleteByID deletes a product by its ID.
func (h *Handler) DeleteByID(w http.ResponseWriter, r *http.Request) {
	mLogger := h.loggerWithReqID(r)
	id, ok := web.ParseID(w, r, mLogger)
	if !ok {
		return
	}
	version, ok := web.ParseValidateGte(r, w, mLogger, "version", 1)
	if !ok {
		return
	}
	mLogger.DebugContext(r.Context(), "Received request to delete product", "ID", id, "Version", version)
	if err := h.service.DeleteByID(r.Context(), id, version); err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			mLogger.WarnContext(r.Context(), "Product not found for deletion", "ID", id)
			web.RespondError(w, mLogger, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		mLogger.ErrorContext(r.Context(), "Error deleting product", "ID", id, "error", err)
		web.RespondError(w, mLogger, http.StatusInternalServerError, fmt.Sprintf("Failed to delete product with ID %s", id))
		return

	}
	mLogger.InfoContext(r.Context(), "Product deleted successfully", "ID", id)
	w.WriteHeader(http.StatusNoContent)
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
