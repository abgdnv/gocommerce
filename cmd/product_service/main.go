// Package main implements a simple HTTP server for managing products.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/abgdnv/gocommerce/internal/product/handler"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/abgdnv/gocommerce/internal/product/store"
)

func main() {
	inMemoryStore := store.NewInMemoryStore()
	// Generate some sample products
	_, _ = inMemoryStore.CreateProduct("Sample 1", 1000, 10)
	_, _ = inMemoryStore.CreateProduct("Sample 2", 2000, 20)
	_, _ = inMemoryStore.CreateProduct("Sample 3", 3000, 30)
	_, _ = inMemoryStore.CreateProduct("Sample 4", 4000, 40)
	_, _ = inMemoryStore.CreateProduct("Sample 5", 5000, 50)

	pService := service.NewService(inMemoryStore)

	pApi := handler.NewAPI(pService)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/products/{id}", pApi.ProductsGetById)
	mux.HandleFunc("GET /api/v1/products", pApi.ProductsGet)
	mux.HandleFunc("POST /api/v1/products", pApi.ProductsPost)
	mux.HandleFunc("DELETE /api/v1/products/{id}", pApi.ProductDeleteById)
	mux.HandleFunc("/healthz", pApi.HealthCheckHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Graceful shutdown handling
	idleConnectionsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		fmt.Println("Server is shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server Shutdown: %v\n", err)
		}
		close(idleConnectionsClosed)
	}()

	log.Print("Starting server on :8080")

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server failed to start: %v", err)
	}
	// Wait for the server to shut down gracefully
	<-idleConnectionsClosed
}
