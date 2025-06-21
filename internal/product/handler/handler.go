// Package handler provides HTTP handlers for product-related operations.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	producterrors "github.com/abgdnv/gocommerce/internal/product/errors"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// ProductAPI defines HTTP handlers for product-related endpoints.
type ProductAPI interface {
	FindByID(w http.ResponseWriter, r *http.Request)
	FindAll(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	UpdateStock(w http.ResponseWriter, r *http.Request)
	DeleteByID(w http.ResponseWriter, r *http.Request)

	HealthCheck(w http.ResponseWriter, r *http.Request)
}

type api struct {
	service  service.ProductService
	validate *validator.Validate
}

// NewAPI creates a new instance of ProductAPI with the provided service.
func NewAPI(service service.ProductService) ProductAPI {
	return &api{
		service:  service,
		validate: validator.New(),
	}
}

// FindByID retrieves a product by its ID.
func (a *api) FindByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := a.service.FindByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve product with ID %s", id))
		return
	}
	log.Printf("Retrieved Product ID: %s, Name: %s", t.ID, t.Name)
	respondJSON(w, http.StatusOK, t)

}

// FindAll retrieves a list of all products.
func (a *api) FindAll(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseUrlParam(r, w, "limit")
	if !ok {
		return
	}
	offset, ok := parseUrlParam(r, w, "offset")
	if !ok {
		return
	}

	list, err := a.service.FindAll(r.Context(), offset, limit)
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	respondJSON(w, http.StatusOK, *list)
}

// Create handles the creation of a new product.
func (a *api) Create(w http.ResponseWriter, r *http.Request) {
	var productCreateDto service.ProductCreateDto
	if err := json.NewDecoder(r.Body).Decode(&productCreateDto); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := a.validate.Struct(productCreateDto); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			// If the error is a validation error, we can extract field-specific errors.
			errorResponse := make(map[string]string)
			for _, fieldErr := range validationErrors {
				// fieldErr.Tag() returns "required", "max", etc.
				errorResponse[fieldErr.Field()] = "failed on rule: " + fieldErr.Tag()
			}
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"validation_errors": errorResponse})
			return
		}
		// If it's not a validation error, we can return a generic error.
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	newProduct, err := a.service.Create(r.Context(), productCreateDto)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create product")
		return
	}
	log.Printf("Created Product ID: %s, Name: %s", newProduct.ID, newProduct.Name)
	respondJSON(w, http.StatusCreated, newProduct)
}

func (a *api) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var productDTO service.ProductDto
	if err := json.NewDecoder(r.Body).Decode(&productDTO); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := a.validate.Struct(productDTO); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			errorResponse := make(map[string]string)
			for _, fieldErr := range validationErrors {
				errorResponse[fieldErr.Field()] = "failed on rule: " + fieldErr.Tag()
			}
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"validation_errors": errorResponse})
			return
		}
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	productDTO.ID = id.String()

	updated, err := a.service.Update(r.Context(), productDTO)
	if err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		log.Printf("Error updating product: %v", err)
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update product with ID %s", id))
		return
	}
	log.Printf("Updated Product ID: %s, Name: %s", updated.ID, updated.Name)
	respondJSON(w, http.StatusOK, updated)
}

func (a *api) UpdateStock(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var stockUpdateDTO service.StockUpdateDto
	if err := json.NewDecoder(r.Body).Decode(&stockUpdateDTO); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := a.validate.Struct(stockUpdateDTO); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			errorResponse := make(map[string]string)
			for _, fieldErr := range validationErrors {
				errorResponse[fieldErr.Field()] = "failed on rule: " + fieldErr.Tag()
			}
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{"validation_errors": errorResponse})
			return
		}
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updated, err := a.service.UpdateStock(r.Context(), id, stockUpdateDTO.Stock, stockUpdateDTO.Version)
	if err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		log.Printf("Error updating product stock: %v", err)
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update stock for product with ID %s", id))
		return
	}
	log.Printf("Updated Stock for Product ID: %s, New Stock: %d", updated.ID, updated.Stock)
	respondJSON(w, http.StatusOK, updated)
}

// DeleteByID deletes a product by its ID.
func (a *api) DeleteByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	version, ok := parseUrlParam(r, w, "version")
	if !ok {
		return
	}

	if err := a.service.DeleteByID(r.Context(), id, version); err != nil {
		if errors.Is(err, producterrors.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete product with ID %s", id))
		return

	}
	log.Printf("Deleted Product ID: %s", id)
	w.WriteHeader(http.StatusNoContent)
}

// HealthCheck is a simple health check endpoint.
func (a *api) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	// Handle nil payload
	if payload == nil {
		w.WriteHeader(status)
		return
	}

	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(response)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// parseID extracts and validates the product ID from the request path. Returns the ID and a boolean indicating success.
func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	pathValueID := r.PathValue("id")
	id, err := uuid.Parse(pathValueID)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid product ID: %s", pathValueID))
		return uuid.UUID{}, false
	}
	return id, true
}

func parseUrlParam(r *http.Request, w http.ResponseWriter, key string) (int32, bool) {
	value := r.URL.Query().Get(key)
	if value == "" {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("%s url parameter is required", key))
		return 0, false // Return false if the parameter is not present
	}
	intValue, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid %s number: %s", key, value))
		return 0, false
	}
	return int32(intValue), true
}
