package store

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	perrors "github.com/abgdnv/gocommerce/internal/product/errors"
	"github.com/abgdnv/gocommerce/internal/product/store/db"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const skipIntegrationTests = "PRODUCT_SVC_SKIP_INTEGRATION_TESTS"

// ProductStoreSuite is a test suite for the ProductStore implementation.
type ProductStoreSuite struct {
	suite.Suite                             // Embedding testify's suite for structured testing
	pgContainer *postgres.PostgresContainer // PostgreSQL container for E2E tests
	dbPool      *pgxpool.Pool               // PostgreSQL connection pool for E2E tests
	store       ProductStore                //
	logger      *slog.Logger                // Logger for the test suite
	ctx         context.Context             // Context for the test suite, used for cancellation and timeouts
}

// SetupSuite initializes the test suite by setting up a PostgreSQL container,
func (s *ProductStoreSuite) SetupSuite() {
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
	require.NoError(s.T(), err, "Failed to create pgxpool")

	// 3.1 Ping the database to ensure the connection is established
	for i := range 10 {
		s.logger.Info("Pinging PostgreSQL database", "attempt", i+1)
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
	migrationsPath := filepath.Join(wd, "..", "migrations")
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

	s.store = NewPgStore(s.dbPool)
	s.logger.Info("Initialization complete for ProductStoreSuite")
}

// TearDownSuite cleans up resources after all tests in the suite have run.
func (s *ProductStoreSuite) TearDownSuite() {
	s.logger.Info("Tearing down suite...")
	if s.dbPool != nil {
		s.dbPool.Close()
		s.logger.Info("DB pool closed.")
	}
	if s.pgContainer != nil {
		s.logger.Info("Terminating PostgreSQL container...")
		err := s.pgContainer.Terminate(s.ctx)
		if err != nil {
			s.logger.Warn("failed to terminate PostgreSQL container", "error", err)
		} else {
			s.logger.Info("PostgreSQL container terminated.")
		}
	}
}

// SetupTest prepares the database for each test by truncating the products table.
func (s *ProductStoreSuite) SetupTest() {
	_, err := s.dbPool.Exec(s.ctx, "TRUNCATE TABLE products RESTART IDENTITY CASCADE")
	require.NoError(s.T(), err, "Failed to truncate products table")
}

// TestProductStoreIntegration runs the ProductStore integration tests.
func TestProductStoreIntegration(t *testing.T) {
	// Skip integration tests if the environment variable is set
	if os.Getenv(skipIntegrationTests) == "1" {
		t.Skip("Skipping integration tests based on " + skipIntegrationTests + " env var")
	}
	// Run the test suite
	suite.Run(t, new(ProductStoreSuite))
}

// createTestProduct is a helper function to create a product for testing purposes.
func (s *ProductStoreSuite) createTestProduct(name string, price int64, stock int32) *db.Product {
	s.T().Helper()
	product, err := s.store.Create(s.ctx, name, price, stock)
	require.NoError(s.T(), err, "createTestProduct helper failed to create product")
	return product
}

func (s *ProductStoreSuite) TestCreateAndFindByID() {
	// 1. Create a new product
	toCreate := db.CreateParams{
		Name:          "Apple Iphone 15 Pro",
		Price:         59900,
		StockQuantity: 100,
	}
	created := s.createTestProduct(toCreate.Name, toCreate.Price, toCreate.StockQuantity)

	// 2. Check that the product was created successfully
	require.NotZero(s.T(), created.ID, "Created product ID should not be zero")
	require.Equal(s.T(), toCreate.Name, created.Name)
	require.Equal(s.T(), toCreate.Price, created.Price)
	require.Equal(s.T(), toCreate.StockQuantity, created.StockQuantity)
	require.NotZero(s.T(), *created.CreatedAt, "CreatedAt should be set")

	// 3. Fetch the product by ID
	fetched, err := s.store.FindByID(s.ctx, created.ID)

	// 4. Check that the fetched product matches the created product
	require.NoError(s.T(), err, "FindByID should not return an error")
	require.Equal(s.T(), created.ID, fetched.ID)
	require.Equal(s.T(), created.Name, fetched.Name)
	require.Equal(s.T(), created.Price, fetched.Price)
	require.Equal(s.T(), created.StockQuantity, fetched.StockQuantity)
	require.WithinDuration(s.T(), *created.CreatedAt, *fetched.CreatedAt, time.Second)
}

func (s *ProductStoreSuite) TestFindByID_NotFound() {
	// Attempt to fetch a product that does not exist
	_, err := s.store.FindByID(s.ctx, uuid.New())
	// Check that the error is ErrProductNotFound
	require.ErrorIs(s.T(), err, perrors.ErrProductNotFound, "Expected ErrProductNotFound for non-existent product")
}

func (s *ProductStoreSuite) TestListProducts() {

	s.createTestProduct("Product A", 100, 10)
	s.createTestProduct("Product B", 200, 20)

	products, err := s.store.FindAll(s.ctx, 0, 10)

	require.NoError(s.T(), err)
	require.Len(s.T(), *products, 2, "Should retrieve 2 products")
	assert.Equal(s.T(), "Product B", (*products)[0].Name)
	assert.Equal(s.T(), "Product A", (*products)[1].Name)
}

func (s *ProductStoreSuite) TestUpdateProduct() {
	// Create a product to update
	created := s.createTestProduct("Samsung Galaxy S23", 69900, 50)

	// Update the product's details
	toUpdate := db.UpdateParams{
		ID:            created.ID,
		Name:          "Samsung Galaxy S23 Ultra",
		Price:         79900,
		StockQuantity: 30,
		Version:       created.Version,
	}
	updated, err := s.store.Update(s.ctx, toUpdate.ID, toUpdate.Name, toUpdate.Price, toUpdate.StockQuantity, toUpdate.Version)
	require.NoError(s.T(), err, "Update should not return an error")

	// Check that the updated product matches the new details
	require.Equal(s.T(), toUpdate.ID, updated.ID)
	require.Equal(s.T(), toUpdate.Name, updated.Name)
	require.Equal(s.T(), toUpdate.Price, updated.Price)
	require.Equal(s.T(), toUpdate.StockQuantity, updated.StockQuantity)
	require.Greater(s.T(), updated.Version, created.Version, "Version should be incremented after update")
}

func (s *ProductStoreSuite) TestUpdateProduct_NotFound() {
	// Attempt to update a product that does not exist
	nonExistentID := uuid.New()
	toUpdate := db.UpdateParams{
		ID:            nonExistentID,
		Name:          "Non-existent Product",
		Price:         99999,
		StockQuantity: 0,
		Version:       1,
	}
	_, err := s.store.Update(s.ctx, toUpdate.ID, toUpdate.Name, toUpdate.Price, toUpdate.StockQuantity, toUpdate.Version)
	require.ErrorIs(s.T(), err, perrors.ErrProductNotFound, "Expected ErrProductNotFound for non-existent product")
}

func (s *ProductStoreSuite) TestUpdateProduct_WrongVersion() {
	// Create a product to update
	created := s.createTestProduct("Sony Xperia 1 V", 89900, 15)

	// Attempt to update the product with an incorrect version
	toUpdate := db.UpdateParams{
		ID:            created.ID,
		Name:          "Sony Xperia 1 V Pro",
		Price:         94900,
		StockQuantity: 10,
		Version:       created.Version + 1, // Incrementing the version to simulate a conflict
	}
	_, err := s.store.Update(s.ctx, toUpdate.ID, toUpdate.Name, toUpdate.Price, toUpdate.StockQuantity, toUpdate.Version)
	require.ErrorIs(s.T(), err, perrors.ErrProductNotFound, "Expected ErrProductNotFound for wrong version")
}

func (s *ProductStoreSuite) TestUpdateStock() {
	// Create a product to update stock
	created := s.createTestProduct("Google Pixel 8", 59900, 20)

	// Update the product's stock
	newStock := int32(15)
	updated, err := s.store.UpdateStock(s.ctx, created.ID, newStock, created.Version)
	require.NoError(s.T(), err, "UpdateStock should not return an error")

	// Check that the updated product has the new stock quantity
	require.Equal(s.T(), created.ID, updated.ID)
	require.Equal(s.T(), newStock, updated.StockQuantity)
	require.Greater(s.T(), updated.Version, created.Version, "Version should be incremented after stock update")
}

func (s *ProductStoreSuite) TestUpdateStock_NotFound() {
	// Attempt to update stock for a product that does not exist
	nonExistentID := uuid.New()
	newStock := int32(10)
	_, err := s.store.UpdateStock(s.ctx, nonExistentID, newStock, 1)
	require.ErrorIs(s.T(), err, perrors.ErrProductNotFound, "Expected ErrProductNotFound for non-existent product")
}

func (s *ProductStoreSuite) TestUpdateStock_WrongVersion() {
	// Create a product to update stock
	created := s.createTestProduct("Xiaomi 13 Pro", 74900, 30)

	// Attempt to update stock with an incorrect version
	newStock := int32(25)
	wrongVersion := created.Version + 1 // Incrementing the version to simulate a conflict
	_, err := s.store.UpdateStock(s.ctx, created.ID, newStock, wrongVersion)
	require.ErrorIs(s.T(), err, perrors.ErrProductNotFound, "Expected ErrProductNotFound for wrong version")
}

func (s *ProductStoreSuite) TestDeleteByID() {
	// Create a product to delete
	created := s.createTestProduct("OnePlus 11", 54900, 25)

	// Delete the product by ID
	err := s.store.DeleteByID(s.ctx, created.ID, created.Version)
	require.NoError(s.T(), err, "DeleteByID should not return an error")

	// Attempt to fetch the deleted product
	_, err = s.store.FindByID(s.ctx, created.ID)
	require.ErrorIs(s.T(), err, perrors.ErrProductNotFound, "Expected ErrProductNotFound for deleted product")
}

func (s *ProductStoreSuite) TestDeleteByID_NotFound() {
	// Attempt to delete a product that does not exist
	nonExistentID := uuid.New()
	err := s.store.DeleteByID(s.ctx, nonExistentID, 1)
	require.ErrorIs(s.T(), err, perrors.ErrProductNotFound, "Expected ErrProductNotFound for non-existent product")
}

func (s *ProductStoreSuite) TestDeleteByID_WrongVersion() {
	// Create a product to delete
	created := s.createTestProduct("Oppo Find N2", 79900, 10)

	// Attempt to delete the product with an incorrect version
	wrongVersion := created.Version + 1 // Incrementing the version to simulate a conflict
	err := s.store.DeleteByID(s.ctx, created.ID, wrongVersion)
	require.ErrorIs(s.T(), err, perrors.ErrProductNotFound, "Expected ErrProductNotFound for wrong version")
}
