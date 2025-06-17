// Package handler provides HTTP handlers for product-related operations.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/abgdnv/gocommerce/internal/product/errorsProduct"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/go-playground/validator/v10"
)

type API struct {
	service  service.ProductServiceContract
	validate *validator.Validate
}

func NewAPI(service service.ProductServiceContract) *API {
	return &API{
		service:  service,
		validate: validator.New(),
	}
}

func (a *API) ProductsPost(w http.ResponseWriter, r *http.Request) {
	var productDTO service.ProductDTO
	if err := json.NewDecoder(r.Body).Decode(&productDTO); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := a.validate.Struct(productDTO); err != nil {
		respondError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	newProduct, err := a.service.CreateProduct(productDTO)
	if err != nil {
		log.Printf("Error creating product: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create product")
		return
	}
	log.Printf("Created Product ID: %s, Name: %s", newProduct.ID, newProduct.Name)
	respondJSON(w, http.StatusCreated, newProduct)
}

func (a *API) ProductsGet(w http.ResponseWriter, r *http.Request) {
	list, err := a.service.GetProducts()
	if err != nil {
		log.Printf("Error fetching products: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	respondJSON(w, http.StatusOK, list)
}

func (a *API) ProductsGetById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := a.service.GetProductByID(id)
	if err != nil {
		if errors.Is(err, errorsProduct.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve product with ID %s", id))
		return
	}
	log.Printf("Retrieved Product ID: %s, Name: %s", t.ID, t.Name)
	respondJSON(w, http.StatusOK, t)

}

func (a *API) ProductDeleteById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := a.service.DeleteProductByID(id)
	if err != nil {
		if errors.Is(err, errorsProduct.ErrProductNotFound) {
			respondError(w, http.StatusNotFound, fmt.Sprintf("Product with ID %s not found", id))
			return
		}
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete product with ID %s", id))
		return

	}
	log.Printf("Deleted Product ID: %s", id)
	w.WriteHeader(http.StatusNoContent)
}

// HealthCheckHandler is a simple health check endpoint.
func (a *API) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
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
