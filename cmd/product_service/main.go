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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	inMemoryStore := store.NewInMemoryStore()
	// Generate some sample products
	_, _ = inMemoryStore.Create("Sample 1", 1000, 10)
	_, _ = inMemoryStore.Create("Sample 2", 2000, 20)
	_, _ = inMemoryStore.Create("Sample 3", 3000, 30)
	_, _ = inMemoryStore.Create("Sample 4", 4000, 40)
	_, _ = inMemoryStore.Create("Sample 5", 5000, 50)

	pService := service.NewService(inMemoryStore)

	pApi := handler.NewAPI(pService)

	mux := chi.NewRouter()
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Timeout(30 * time.Second))

	mux.Route("/api/v1/products", func(r chi.Router) {
		r.Get("/", pApi.FindAll)
		r.Post("/", pApi.Create)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", pApi.FindByID)
			r.Delete("/", pApi.DeleteByID)
		})
	})

	mux.Get("/healthz", pApi.HealthCheck)

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
