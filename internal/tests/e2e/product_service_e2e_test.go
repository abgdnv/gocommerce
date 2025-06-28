// Package e2e provides end-to-end tests for the ProductService application.
// The suite leverages `testcontainers-go` to spin up a real PostgreSQL instance in a Docker container,
// ensuring tests run against a production-like environment. It uses `testify/suite` for better structure
// and lifecycle management (`SetupSuite`, `TearDownSuite`, `SetupTest`).
//
// Key features of the test suite:
//   - A PostgreSQL container is started and database migrations are applied before tests run.
//   - The actual application handler is run in an `httptest.Server`.
//   - Table-driven tests are used to cover a wide range of scenarios for all API endpoints (GET, POST, PUT, DELETE).
//   - Each test case is fully isolated by truncating the database tables before it runs.
//   - Test coverage includes:
//   - Happy path CRUD operations.
//   - Pagination and filtering (offset, limit).
//   - Input validation for invalid data (e.g., negative price, empty name).
//   - Optimistic locking checks using the 'version' field.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abgdnv/gocommerce/internal/config"
	"github.com/abgdnv/gocommerce/internal/product/app"
	"github.com/abgdnv/gocommerce/internal/product/service"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// skipE2ETests is the environment variable that can be set to skip E2E tests.
const skipE2ETests = "PRODUCT_SVC_SKIP_E2E_TESTS"

// productURL is the base URL for the ProductService API.
const productURL = "/api/v1/products"

// ProductServiceE2ESuite is a test suite for end-to-end tests of the ProductService.
type ProductServiceE2ESuite struct {
	suite.Suite                             // Embedding testify's suite for structured testing
	pgContainer *postgres.PostgresContainer // PostgreSQL container for E2E tests
	dbPool      *pgxpool.Pool               // PostgreSQL connection pool for E2E tests
	server      *httptest.Server            // HTTP server for the ProductService application
	httpClient  *http.Client                // HTTP client for making requests to the server
	appCfg      *config.Config              // Application configuration for tests
	logger      *slog.Logger                // Logger for the test suite
	ctx         context.Context             // Context for the test suite, used for cancellation and timeouts
}

// testConfig creates a configuration for the ProductService application (only HTTPServer settings).
func testConfig() *config.Config {
	var cfg config.Config

	// HTTPServer settings
	cfg.HTTPServer.Port = 0                 // httptest.Server will assign a random port
	cfg.HTTPServer.MaxHeaderBytes = 1 << 20 // 1 MB
	// Set timeouts for the HTTP server (increased for E2E tests debugging)
	cfg.HTTPServer.Timeout.Read = 10 * time.Minute
	cfg.HTTPServer.Timeout.Write = 10 * time.Minute
	cfg.HTTPServer.Timeout.Idle = 60 * time.Minute
	cfg.HTTPServer.Timeout.ReadHeader = 5 * time.Minute

	return &cfg

}

// SetupSuite initializes the test suite by setting up the PostgreSQL container, database connection, and application configuration.
func (s *ProductServiceE2ESuite) SetupSuite() {
	s.ctx = context.Background()
	var err error
	s.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	dbName := "products"
	dbUser := "user"
	dbPassword := "password"

	// 1. Start a PostgreSQL container with the specified configuration. Wait for the container to be ready.
	s.pgContainer, err = postgres.Run(s.ctx,
		"postgres:17.5-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		// Wait for a specific log message indicating the database service is ready.
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Minute),
		),
		// Ensure the container is ready to accept connections on the default PostgreSQL port.
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
		),
	)
	require.NoError(s.T(), err, "Failed to run PostgreSQL container")

	// 2. Get the connection string from the container
	connStr, err := s.pgContainer.ConnectionString(s.ctx, "sslmode=disable")
	require.NoError(s.T(), err, "Failed to get connection string from container")

	// 3. create a new pgxpool instance using the connection string
	s.dbPool, err = pgxpool.New(s.ctx, connStr)
	require.NoError(s.T(), err, "Failed to create pgx pool")

	// 3.1 Ping the database to ensure the connection is established
	for i := range 10 {
		s.logger.Info("Pinging E2E PostgreSQL database", "attempt", i+1)
		err = s.dbPool.Ping(s.ctx)
		if err == nil {
			break
		}
		time.Sleep(time.Second * 2)
	}
	require.NoError(s.T(), err, "Failed to connect to PostgreSQL after retries")

	// 4. Database migration
	// Build path to migrations directory
	wd, _ := os.Getwd()
	migrationsPath := filepath.Join(wd, "..", "..", "product", "migrations")
	sourceURL := "file://" + migrationsPath
	// Create a new migrate instance with the source URL and connection string
	m, err := migrate.New(sourceURL, connStr)
	require.NoError(s.T(), err, "Failed to create migrate instance")
	// Apply all available migrations
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_, _ = m.Close()
		require.NoError(s.T(), err, "Failed to apply migrations")
	}
	s.logger.Info("Migrations applied for E2E tests")

	// 5. Create the application configuration for tests
	s.appCfg = testConfig()

	// 6. Set up the application configuration
	appHandler, err := app.SetupApplication(s.appCfg, s.dbPool, s.logger)
	require.NoError(s.T(), err, "Failed to setup application for E2E")

	s.server = httptest.NewServer(appHandler)
	s.httpClient = s.server.Client() // Use the httptest server's client for requests
	s.logger.Info("E2E test server started", "url", s.server.URL)
}

// TearDownSuite cleans up resources after all tests in the suite have run.
func (s *ProductServiceE2ESuite) TearDownSuite() {
	s.logger.Info("Tearing down E2E suite...")
	if s.server != nil {
		s.server.Close()
		s.logger.Info("E2E test server closed.")
	}
	if s.dbPool != nil {
		s.dbPool.Close()
		s.logger.Info("E2E DB pool closed.")
	}
	if s.pgContainer != nil {
		s.logger.Info("Terminating E2E PostgreSQL container...")
		err := s.pgContainer.Terminate(s.ctx)
		if err != nil {
			s.logger.Warn("Failed to terminate E2E PostgreSQL container", "error", err)
		} else {
			s.logger.Info("E2E PostgreSQL container terminated.")
		}
	}
}

// SetupTest prepares the database for each test by truncating the products table.
func (s *ProductServiceE2ESuite) SetupTest() {
	_, err := s.dbPool.Exec(s.ctx, "TRUNCATE TABLE products RESTART IDENTITY CASCADE")
	require.NoError(s.T(), err, "Failed to truncate products table")
}

// TestProductStoreIntegration runs the ProductStore integration tests.
func TestProductServiceE2E(t *testing.T) {
	// Skip integration tests if the environment variable is set
	if os.Getenv(skipE2ETests) == "1" {
		t.Skip("Skipping integration tests based on " + skipE2ETests + " env var")
	}
	// Run the test suite
	suite.Run(t, new(ProductServiceE2ESuite))
}

// --------------------------------------------------------------------------
// ---------- Payload structures and Helper methods for E2E tests -----------
// --------------------------------------------------------------------------

// createProductPayload is a struct used to represent the payload for creating a product.
type createProductPayload struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
	Stock int32  `json:"stock"`
}

// updateProductPayload is a struct used to represent the payload for updating a product.
type updateProductPayload struct {
	Name    string `json:"name"`
	Price   int64  `json:"price"`
	Stock   int32  `json:"stock"`
	Version int32  `json:"version"`
}

// updateStockPayload is a struct used to represent the payload for updating the stock of a product.
type updateStockPayload struct {
	Stock   int32 `json:"stock"`
	Version int32 `json:"version"`
}

// FindByID is a helper method to fetch a product by its ID from the service.
// Returns the ProductDto and the HTTP status code.
func (s *ProductServiceE2ESuite) FindByID(ID string) (service.ProductDto, int) {
	s.T().Helper()
	getURL := s.server.URL + productURL + "/" + ID
	return s.doAndDecodeProduct(http.MethodGet, getURL, nil)
}

// FindAllProducts is a helper method to fetch all products from the service.
// Returns a slice of ProductDto and the HTTP status code.
func (s *ProductServiceE2ESuite) FindAllProducts(offset, limit int) ([]service.ProductDto, int) {
	s.T().Helper()
	url := s.server.URL + productURL + fmt.Sprintf("?offset=%d&limit=%d", offset, limit)
	return s.doAndDecodeProductList(http.MethodGet, url, nil)
}

// createProduct is a helper method to create a product and decode the response into a ProductDto.
// Returns the created ProductDto and the HTTP status code.
func (s *ProductServiceE2ESuite) createProduct(payload createProductPayload) (service.ProductDto, int) {
	s.T().Helper()
	createURL := s.server.URL + productURL
	return s.doAndDecodeProduct(http.MethodPost, createURL, payload)
}

// updateProduct is a helper method to update a product and decode the response into a ProductDto.
// Returns the updated ProductDto and the HTTP status code.
func (s *ProductServiceE2ESuite) updateProduct(productID string, payload updateProductPayload) (service.ProductDto, int) {
	s.T().Helper()
	updateURL := fmt.Sprintf("%s/%s", s.server.URL+productURL, productID)
	return s.doAndDecodeProduct(http.MethodPut, updateURL, payload)
}

// updateStock is a helper method to update the stock of a product and decode the response into a ProductDto.
// Returns the updated ProductDto and the HTTP status code.
func (s *ProductServiceE2ESuite) updateStock(productID string, payload updateStockPayload) (service.ProductDto, int) {
	s.T().Helper()
	updateStockURL := fmt.Sprintf("%s/%s/stock", s.server.URL+productURL, productID)
	return s.doAndDecodeProduct(http.MethodPut, updateStockURL, payload)
}

// deleteByID is a helper method to delete a product by its ID and version.
// Returns the HTTP status code.
func (s *ProductServiceE2ESuite) deleteByID(productID string, version int32) int {
	s.T().Helper()
	deleteURL := fmt.Sprintf("%s/%s?version=%d", s.server.URL+productURL, productID, version)
	_, statusCode := s.doRequest(http.MethodDelete, deleteURL, nil)
	return statusCode
}

// doAndDecodeProduct is a helper method to make an HTTP request to the product service and decode the response into a ProductDto.
// Returns the ProductDto and the HTTP status code.
func (s *ProductServiceE2ESuite) doAndDecodeProduct(method, url string, payload any) (service.ProductDto, int) {
	s.T().Helper()
	bodyBytes, statusCode := s.doRequest(method, url, payload)

	var product service.ProductDto
	if statusCode == http.StatusOK || statusCode == http.StatusCreated {
		product = s.decodeProductResponse(bodyBytes)
	}
	return product, statusCode
}

// doAndDecodeProductList is a helper method to make an HTTP request to the product service and decode the response into a slice of ProductDto.
// Returns the slice of ProductDto and the HTTP status code.
func (s *ProductServiceE2ESuite) doAndDecodeProductList(method, url string, payload any) ([]service.ProductDto, int) {
	s.T().Helper()
	bodyBytes, statusCode := s.doRequest(method, url, payload)

	var products []service.ProductDto
	if statusCode == http.StatusOK {
		products = s.decodeProductListResponse(bodyBytes)
	}
	return products, statusCode
}

// doRequest is a helper method to make an HTTP request to the product service
// Returns the response body as a byte slice and the HTTP status code.
func (s *ProductServiceE2ESuite) doRequest(method, url string, payload any) ([]byte, int) {
	s.T().Helper()
	var body io.Reader
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		require.NoError(s.T(), err)
		body = bytes.NewBuffer(payloadBytes)
	}

	req, err := http.NewRequestWithContext(s.ctx, method, url, body)
	require.NoError(s.T(), err, "Failed to create HTTP request")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err, "HTTP request failed")
	defer func() {
		err := resp.Body.Close()
		require.NoError(s.T(), err, "Failed to close response body")
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err, "Failed to read response body")

	return bodyBytes, resp.StatusCode
}

// decodeProductResponse is a helper method to decode the response body into a ProductDto.
// Returns the decoded ProductDto.
func (s *ProductServiceE2ESuite) decodeProductResponse(bodyBytes []byte) service.ProductDto {
	s.T().Helper()
	var product service.ProductDto
	err := json.Unmarshal(bodyBytes, &product)
	require.NoError(s.T(), err, "Failed to decode product response")

	return product
}

// decodeProductListResponse is a helper method to decode the response body into a slice of ProductDto.
// Returns the decoded slice of ProductDto.
func (s *ProductServiceE2ESuite) decodeProductListResponse(bodyBytes []byte) []service.ProductDto {
	s.T().Helper()
	var products []service.ProductDto
	err := json.Unmarshal(bodyBytes, &products)
	require.NoError(s.T(), err, "Failed to decode product list response")
	return products
}

// --------------------------------------------------------------
// ---------------------- E2E test methods ----------------------
// --------------------------------------------------------------

func (s *ProductServiceE2ESuite) TestFindByID_NotFound_E2E() {
	s.T().Run("Find Product By ID - Not Found", func(t *testing.T) {
		s.SetupTest()
		// given
		nonExistentID := uuid.New().String()

		// when
		_, statusCode := s.FindByID(nonExistentID)

		// then
		require.Equal(t, http.StatusNotFound, statusCode)
	})
}

func (s *ProductServiceE2ESuite) TestFindAll_E2E() {
	testCases := []struct {
		name           string
		createPayload  createProductPayload
		amount         int
		offset         int
		limit          int
		expectedCode   int
		expectedAmount int
	}{
		{
			name:           "Find All Products - No Products",
			createPayload:  createProductPayload{},
			amount:         0,
			offset:         0,
			limit:          10,
			expectedCode:   http.StatusOK,
			expectedAmount: 0,
		},
		{
			name:           "Find All Products - One Product",
			createPayload:  createProductPayload{"Apple iPhone 15 Pro Max", int64(59900), int32(100)},
			amount:         1,
			offset:         0,
			limit:          10,
			expectedCode:   http.StatusOK,
			expectedAmount: 1,
		},
		{
			name:           "Find All Products - Multiple Products",
			createPayload:  createProductPayload{"Samsung Galaxy S23 Ultra", int64(119900), int32(50)},
			amount:         5,
			offset:         0,
			limit:          10,
			expectedCode:   http.StatusOK,
			expectedAmount: 5,
		},
		{
			name:           "Find All Products - Limit",
			createPayload:  createProductPayload{"Google Pixel 8 Pro", int64(89900), int32(75)},
			amount:         5,
			offset:         0,
			limit:          3,
			expectedCode:   http.StatusOK,
			expectedAmount: 3,
		},
		{
			name:           "Find All Products - Offset",
			createPayload:  createProductPayload{"Google Pixel 8 Pro", int64(89900), int32(75)},
			amount:         5,
			offset:         3,
			limit:          10,
			expectedCode:   http.StatusOK,
			expectedAmount: 2,
		},
		{
			name:         "Find All Products - Validate Offset",
			amount:       0,
			offset:       -1,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Find All Products - Validate Limit",
			amount:       0,
			limit:        -1,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			// given
			for i := 0; i < tc.amount; i++ {
				_, statusCode := s.createProduct(tc.createPayload)
				require.Equal(t, http.StatusCreated, statusCode, "Expected HTTP 201 Created")
			}

			// when
			products, statusCode := s.FindAllProducts(tc.offset, tc.limit)

			// then
			require.Equal(t, tc.expectedCode, statusCode, "Expected HTTP %d", tc.expectedCode)
			require.Len(t, products, tc.expectedAmount, "Expected %d products", tc.expectedAmount)
		})
	}
}

// TestCreateProduct_E2E tests the creation of products with various payloads.
func (s *ProductServiceE2ESuite) TestCreateProduct_E2E() {
	testCases := []struct {
		name            string
		payload         createProductPayload
		expectedCode    int
		expectedProduct service.ProductDto
	}{
		{
			name:            "Create Product - Empty Name",
			payload:         createProductPayload{Name: "", Price: 100, Stock: 10},
			expectedCode:    http.StatusBadRequest,
			expectedProduct: service.ProductDto{},
		},
		{
			name:            "Create Product - Negative Price",
			payload:         createProductPayload{Name: "Test Product", Price: -50, Stock: 10},
			expectedCode:    http.StatusBadRequest,
			expectedProduct: service.ProductDto{},
		},
		{
			name:            "Create Product - Negative Stock",
			payload:         createProductPayload{Name: "Test Product", Price: 100, Stock: -1},
			expectedCode:    http.StatusBadRequest,
			expectedProduct: service.ProductDto{},
		},
		{
			name:            "Create Product - Valid Product",
			payload:         createProductPayload{Name: "Valid Product", Price: 100, Stock: 10},
			expectedCode:    http.StatusCreated,
			expectedProduct: service.ProductDto{Name: "Valid Product", Price: 100, Stock: 10, Version: 1},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			// when
			product, statusCode := s.createProduct(tc.payload)

			// then
			require.Equal(t, tc.expectedCode, statusCode)
			if tc.expectedCode == http.StatusCreated {
				require.NotZero(t, product.ID)
				require.Equal(t, tc.expectedProduct.Name, product.Name)
				require.Equal(t, tc.expectedProduct.Price, product.Price)
				require.Equal(t, tc.expectedProduct.Stock, product.Stock)
				require.Equal(t, tc.expectedProduct.Version, product.Version)

				// Verify that the product can be fetched by ID
				fetchedProduct, statusCode := s.FindByID(product.ID)

				require.Equal(t, http.StatusOK, statusCode)
				require.Equal(t, product.ID, fetchedProduct.ID)
				require.Equal(t, product.Name, fetchedProduct.Name)
				require.Equal(t, product.Price, fetchedProduct.Price)
				require.Equal(t, product.Stock, fetchedProduct.Stock)
				require.Equal(t, product.Version, fetchedProduct.Version)

			}
		})
	}
}

func (s *ProductServiceE2ESuite) TestUpdateProduct_E2E() {

	testCases := []struct {
		name            string
		createPayload   createProductPayload
		updatePayload   updateProductPayload
		expectedCode    int
		expectedProduct service.ProductDto
	}{
		{
			name:            "Update Product - Valid Product",
			createPayload:   createProductPayload{"Valid Product", int64(59900), int32(100)},
			updatePayload:   updateProductPayload{"Valid Product Updated", int64(64900), int32(120), 1},
			expectedCode:    http.StatusOK,
			expectedProduct: service.ProductDto{Name: "Valid Product Updated", Price: 64900, Stock: 120, Version: 2},
		},
		{
			name:            "Update Product - Product with wrong version",
			createPayload:   createProductPayload{"Samsung Galaxy S23 Ultra", int64(119900), int32(50)},
			updatePayload:   updateProductPayload{"Samsung Galaxy S23 Ultra Updated", int64(129900), int32(60), 2},
			expectedCode:    http.StatusNotFound,
			expectedProduct: service.ProductDto{},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			// given
			createdProduct, statusCode := s.createProduct(tc.createPayload)
			require.Equal(t, http.StatusCreated, statusCode)

			// when
			updatedProduct, statusCode := s.updateProduct(createdProduct.ID, tc.updatePayload)

			// then
			require.Equal(t, tc.expectedCode, statusCode)
			if tc.expectedCode == http.StatusOK {
				require.Equal(t, createdProduct.ID, updatedProduct.ID)
				require.Equal(t, tc.expectedProduct.Name, updatedProduct.Name)
				require.Equal(t, tc.expectedProduct.Price, updatedProduct.Price)
				require.Equal(t, tc.expectedProduct.Stock, updatedProduct.Stock)
				require.Equal(t, tc.expectedProduct.Version, updatedProduct.Version)
			}
		})
	}

}

func (s *ProductServiceE2ESuite) TestUpdateStock_E2E() {
	testCases := []struct {
		name          string
		createPayload createProductPayload
		updatePayload updateStockPayload
		version       int32
		expectedCode  int
	}{
		{
			name:          "Update Stock - with valid version",
			createPayload: createProductPayload{"Apple iPhone 15 Pro Max", int64(59900), int32(100)},
			updatePayload: updateStockPayload{int32(150), int32(1)},
			expectedCode:  http.StatusOK,
		},
		{
			name:          "Update Stock - with wrong version",
			createPayload: createProductPayload{"Samsung Galaxy S23 Ultra", int64(119900), int32(50)},
			updatePayload: updateStockPayload{int32(60), int32(2)},
			expectedCode:  http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			// given
			createdProduct, statusCode := s.createProduct(tc.createPayload)
			require.Equal(t, http.StatusCreated, statusCode)

			// when
			updatedProduct, statusCode := s.updateStock(createdProduct.ID, tc.updatePayload)

			// then
			require.Equal(t, tc.expectedCode, statusCode)
			if tc.expectedCode == http.StatusOK {
				require.Equal(t, createdProduct.ID, updatedProduct.ID)
				require.Equal(t, tc.updatePayload.Stock, updatedProduct.Stock)
				require.Equal(t, tc.updatePayload.Version+1, updatedProduct.Version)
			}
		})
	}
}

func (s *ProductServiceE2ESuite) TestDeleteProduct_E2E() {
	// given
	testCases := []struct {
		name         string
		payload      createProductPayload
		version      int32
		expectedCode int
	}{
		{
			name:         "Delete Product - with valid version",
			payload:      createProductPayload{"Apple iPhone 15 Pro Max", int64(59900), int32(100)},
			version:      int32(1),
			expectedCode: http.StatusNoContent,
		},
		{
			name:         "Delete Product - with wrong version",
			payload:      createProductPayload{"Samsung Galaxy S23 Ultra", int64(119900), int32(50)},
			version:      int32(2),
			expectedCode: http.StatusNotFound,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.SetupTest()
			// given
			createdProduct, statusCode := s.createProduct(tc.payload)
			require.Equal(t, http.StatusCreated, statusCode)

			// when
			statusCode = s.deleteByID(createdProduct.ID, tc.version)

			// then
			require.Equal(t, tc.expectedCode, statusCode)
		})
	}
}
