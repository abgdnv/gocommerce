// Package handler provides HTTP handlers for product-related operations.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	producterrors "github.com/abgdnv/gocommerce/internal/product/errors"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/go-playground/validator/v10"
)

// ProductAPI defines HTTP handlers for product-related endpoints.
type ProductAPI interface {
	FindByID(w http.ResponseWriter, r *http.Request)
	FindAll(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
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
	id := r.PathValue("id")
	t, err := a.service.FindByID(id)
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
	list, err := a.service.FindAll()
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	respondJSON(w, http.StatusOK, list)
}

// Create handles the creation of a new product.
func (a *api) Create(w http.ResponseWriter, r *http.Request) {
	var productDTO service.ProductDto
	if err := json.NewDecoder(r.Body).Decode(&productDTO); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := a.validate.Struct(productDTO); err != nil {
		respondError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	newProduct, err := a.service.Create(productDTO)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create product")
		return
	}
	log.Printf("Created Product ID: %s, Name: %s", newProduct.ID, newProduct.Name)
	respondJSON(w, http.StatusCreated, newProduct)
}

// DeleteByID deletes a product by its ID.
func (a *api) DeleteByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.service.DeleteByID(id)
	if err != nil {
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
func (a *api) HealthCheck(w http.ResponseWriter, r *http.Request) {
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
