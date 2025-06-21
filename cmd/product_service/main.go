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

	"github.com/abgdnv/gocommerce/internal/config"
	"github.com/abgdnv/gocommerce/internal/product/handler"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/abgdnv/gocommerce/internal/product/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		log.Fatalf("Error loading configuration: %v", cfgErr)
	}
	log.Printf("Configuration loaded: %v", cfg)

	// Create context with timeout for database connection
	poolCtx, poolCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer poolCancel()

	dbPool, errPool := pgxpool.New(poolCtx, cfg.Database.URL)
	if errPool != nil {
		log.Fatalf("Unable to create connection pool: %v\n", errPool)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(poolCtx); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}
	log.Println("Successfully connected to the database!")

	pService := service.NewService(store.NewPgStore(dbPool))

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
		Addr:              fmt.Sprintf(":%d", cfg.HTTPServer.Port),
		Handler:           mux,
		ReadTimeout:       cfg.HTTPServer.Timeout.Read,
		WriteTimeout:      cfg.HTTPServer.Timeout.Write,
		IdleTimeout:       cfg.HTTPServer.Timeout.Idle,
		ReadHeaderTimeout: cfg.HTTPServer.Timeout.ReadHeader,
		MaxHeaderBytes:    cfg.HTTPServer.MaxHeaderBytes,
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

	log.Printf("Starting server on %s", server.Addr)

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server failed to start: %v", err)
	}
	// Wait for the server to shut down gracefully
	<-idleConnectionsClosed
}
