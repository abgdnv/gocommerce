package store

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	ordererrors "github.com/abgdnv/gocommerce/order_service/internal/errors"
	"github.com/abgdnv/gocommerce/order_service/internal/store/db"
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

const skipIntegrationTests = "ORDER_SVC_SKIP_INTEGRATION_TESTS"

// OrderStoreSuite is a test suite for the OrderStore implementation.
type OrderStoreSuite struct {
	suite.Suite                             // Embedding testify suite for structured testing
	pgContainer *postgres.PostgresContainer // PostgreSQL container for E2E tests
	dbPool      *pgxpool.Pool               // PostgreSQL connection pool for E2E tests
	store       OrderStore                  //
	logger      *slog.Logger                // Logger for the test suite
	ctx         context.Context             // Context for the test suite, used for cancellation and timeouts
}

// SetupSuite initializes the test suite by setting up a PostgreSQL container,
func (s *OrderStoreSuite) SetupSuite() {
	s.ctx = context.Background()
	var err error
	s.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	dbName := "orders_db"
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
	migrationsPath := filepath.Join(wd, "../../../deploy/charts/db-migrations/migrations/order_service")
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
	s.logger.Info("Initialization complete for OrderStoreSuite")
}

// TearDownSuite cleans up resources after all tests in the suite have run.
func (s *OrderStoreSuite) TearDownSuite() {
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

// SetupTest prepares the database for each test by truncating the orders table.
func (s *OrderStoreSuite) SetupTest() {
	_, err := s.dbPool.Exec(s.ctx, "TRUNCATE TABLE orders RESTART IDENTITY CASCADE")
	require.NoError(s.T(), err, "Failed to truncate orders table")
}

// TestOrderStoreIntegration runs the OrderStore integration tests.
func TestOrderStoreIntegration(t *testing.T) {
	// Skip integration tests if the environment variable is set
	if os.Getenv(skipIntegrationTests) == "1" {
		t.Skip("Skipping integration tests based on " + skipIntegrationTests + " env var")
	}
	// Run the test suite
	suite.Run(t, new(OrderStoreSuite))
}

// createTestOrder is a helper function to create an order for testing purposes.
func (s *OrderStoreSuite) createTestOrder(orderParams *db.CreateOrderParams, itemParams *[]db.CreateOrderItemParams) (*db.Order, *[]db.OrderItem, error) {
	s.T().Helper()
	order, items, err := s.store.CreateOrder(s.ctx, orderParams, itemParams)
	require.NoError(s.T(), err, "createTestOrder helper failed to create order")
	return order, items, nil
}

func (s *OrderStoreSuite) TestCreate() {
	s.SetupTest()
	// given
	orderToCreate := db.CreateOrderParams{
		UserID: uuid.New(),
		Status: "PENDING",
	}
	orderItemToCreate := []db.CreateOrderItemParams{{
		ProductID:    uuid.New(),
		Quantity:     2,
		PricePerItem: 1000,
		Price:        2000,
	}}

	// when
	createdOrder, createdItems, err := s.createTestOrder(&orderToCreate, &orderItemToCreate)

	// then
	require.NoError(s.T(), err, "CreateOrder should not return an error")

	require.NotZero(s.T(), createdOrder.ID, "Created order ID should not be zero")
	require.Equal(s.T(), orderToCreate.UserID, createdOrder.UserID)
	require.Equal(s.T(), orderToCreate.Status, createdOrder.Status)
	require.Equal(s.T(), createdOrder.Version, int32(1), "Version should be 1 for newly created order")
	require.NotZero(s.T(), *createdOrder.CreatedAt, "CreatedAt should be set")

	require.Len(s.T(), *createdItems, 1, "Should create one order item")
	require.NotZero(s.T(), (*createdItems)[0].ID, "Created order item ID should not be zero")
	require.Equal(s.T(), orderItemToCreate[0].ProductID, (*createdItems)[0].ProductID)
	require.Equal(s.T(), orderItemToCreate[0].Quantity, (*createdItems)[0].Quantity)
	require.Equal(s.T(), orderItemToCreate[0].PricePerItem, (*createdItems)[0].PricePerItem)
	require.Equal(s.T(), orderItemToCreate[0].Price, (*createdItems)[0].Price)
	require.NotZero(s.T(), *(*createdItems)[0].CreatedAt, "CreatedAt for order item should be set")
}

func (s *OrderStoreSuite) TestFindByID() {
	s.SetupTest()
	// given
	orderToCreate := db.CreateOrderParams{
		UserID: uuid.New(),
		Status: "PENDING",
	}
	orderItemToCreate := []db.CreateOrderItemParams{{
		ProductID:    uuid.New(),
		Quantity:     2,
		PricePerItem: 1000,
		Price:        2000,
	}}
	createdOrder, createdItems, err := s.createTestOrder(&orderToCreate, &orderItemToCreate)
	require.NoError(s.T(), err, "CreateOrder should not return an error")

	// when
	fetchedOrder, fetchedOrderItems, err := s.store.FindByID(s.ctx, createdOrder.ID)

	// then
	require.NoError(s.T(), err, "FindByID should not return an error")

	require.Equal(s.T(), createdOrder.ID, fetchedOrder.ID)
	require.Equal(s.T(), createdOrder.UserID, fetchedOrder.UserID)
	require.Equal(s.T(), createdOrder.Status, fetchedOrder.Status)
	require.WithinDuration(s.T(), *createdOrder.CreatedAt, *fetchedOrder.CreatedAt, time.Second)

	require.Len(s.T(), *fetchedOrderItems, 1, "Should create one order item")
	require.Equal(s.T(), (*createdItems)[0].ID, (*fetchedOrderItems)[0].ID, "Order item ID should match")
	require.Equal(s.T(), (*createdItems)[0].ProductID, (*fetchedOrderItems)[0].ProductID, "Order item ProductID should match")
	require.Equal(s.T(), (*createdItems)[0].Quantity, (*fetchedOrderItems)[0].Quantity, "Order item Quantity should match")
	require.Equal(s.T(), (*createdItems)[0].PricePerItem, (*fetchedOrderItems)[0].PricePerItem, "Order item PricePerItem should match")
	require.Equal(s.T(), (*createdItems)[0].Price, (*fetchedOrderItems)[0].Price, "Order item Price should match")
	require.WithinDuration(s.T(), *(*createdItems)[0].CreatedAt, *(*fetchedOrderItems)[0].CreatedAt, time.Second)

}

func (s *OrderStoreSuite) TestFindByID_NotFound() {
	s.SetupTest()
	// given (no orders created)

	// when
	_, _, err := s.store.FindByID(s.ctx, uuid.New())

	// then
	require.ErrorIs(s.T(), err, ordererrors.ErrOrderNotFound, "Expected ErrOrderNotFound for non-existent order")
}

func (s *OrderStoreSuite) TestListOrders() {

	const statusCompleted = "COMPLETED"
	const statusPending = "PENDING"
	mockUserID := uuid.New()

	testCases := []struct {
		name        string
		findParams  *db.FindOrdersByUserIDParams
		postCheck   func(t *testing.T, order *[]db.Order)
		expectedErr error
	}{
		{
			name: "List with 2 orders",
			findParams: &db.FindOrdersByUserIDParams{
				UserID: mockUserID,
				Offset: 0,
				Limit:  2,
			},
			postCheck: func(t *testing.T, orders *[]db.Order) {
				require.NotNil(t, orders, "Orders should not be nil")
				require.Len(t, *orders, 2, "Should retrieve 2 orders")
				statuses := make(map[string]bool)
				for _, order := range *orders {
					statuses[order.Status] = true
				}
				assert.True(t, statuses[statusCompleted], "Should contain a completed order")
				assert.True(t, statuses[statusPending], "Should contain a pending order")
			},
			expectedErr: nil,
		},
		{
			name: "List with 1 orders",
			findParams: &db.FindOrdersByUserIDParams{
				UserID: mockUserID,
				Offset: 0,
				Limit:  1,
			},
			postCheck: func(t *testing.T, orders *[]db.Order) {
				require.NotNil(t, orders, "Orders should not be nil")
				require.Len(t, *orders, 1, "Should retrieve 1 order")
			},
			expectedErr: nil,
		},
		{
			name: "Wrong user id",
			findParams: &db.FindOrdersByUserIDParams{
				UserID: uuid.New(), // Non-existent user ID
				Offset: 0,
				Limit:  10,
			},
			postCheck: func(t *testing.T, orders *[]db.Order) {
				require.NotNil(t, orders, "Orders should not be nil")
				require.Len(t, *orders, 0, "Should retrieve no orders for non-existent user")
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		// given
		s.SetupTest()
		_, _, err := s.createTestOrder(&db.CreateOrderParams{UserID: mockUserID, Status: statusPending}, &[]db.CreateOrderItemParams{
			{ProductID: uuid.New(), Quantity: 2, PricePerItem: 1000, Price: 2000},
		})
		require.NoError(s.T(), err, "Failed to create first order")
		_, _, err = s.createTestOrder(&db.CreateOrderParams{UserID: mockUserID, Status: statusCompleted}, &[]db.CreateOrderItemParams{
			{ProductID: uuid.New(), Quantity: 1, PricePerItem: 1500, Price: 1500},
		})
		require.NoError(s.T(), err, "Failed to create second order")

		// when
		orders, err := s.store.FindOrdersByUserID(s.ctx, tc.findParams)

		//then
		if tc.expectedErr != nil {
			require.ErrorIs(s.T(), err, tc.expectedErr, "Expected error for test case: "+tc.name)
		} else {
			require.NoError(s.T(), err)
			if tc.postCheck != nil {
				tc.postCheck(s.T(), orders)
			}
		}
	}
}

func (s *OrderStoreSuite) TestUpdateOrder() {

	const statusCompleted = "COMPLETED"
	nonExistentID := uuid.New()

	testCases := []struct {
		name              string
		nonExistedOrderID bool
		incVersion        int32
		expectedErr       error
		postCheck         func(t *testing.T, initial *db.Order, updated *db.Order)
	}{
		{
			name:        "Successful Update",
			expectedErr: nil,
			postCheck: func(t *testing.T, initial *db.Order, updated *db.Order) {
				require.Equal(t, initial.ID, updated.ID)
				require.Equal(t, statusCompleted, updated.Status)
				require.Equal(t, initial.Version+1, updated.Version, "Version should be incremented")
			},
		},
		{
			name:              "Update Non-Existent Order",
			nonExistedOrderID: true,
			expectedErr:       ordererrors.ErrOrderNotFound,
			postCheck:         nil,
		},
		{
			name:        "Update with Wrong Version",
			incVersion:  1,
			expectedErr: ordererrors.ErrOptimisticLock,
			postCheck:   nil,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			// given
			initialOrder, _, err := s.createTestOrder(&db.CreateOrderParams{UserID: uuid.New(), Status: "PENDING"}, &[]db.CreateOrderItemParams{
				{ProductID: uuid.New(), Quantity: 1, PricePerItem: 50000, Price: 50000},
			})
			require.NoError(s.T(), err, "CreateOrder should not return an error")
			input := db.UpdateOrderParams{
				ID:      initialOrder.ID,
				Status:  statusCompleted,
				Version: initialOrder.Version + tc.incVersion,
			}
			if tc.nonExistedOrderID {
				input.ID = nonExistentID
			}

			// when
			updated, err := s.store.Update(s.ctx, &input)

			// then
			if tc.expectedErr != nil {
				require.ErrorIs(s.T(), err, tc.expectedErr)
				require.Nil(s.T(), updated)
			} else {
				require.NoError(s.T(), err, "Update should not return an error")
				require.NotNil(s.T(), updated)
				if tc.postCheck != nil {
					tc.postCheck(s.T(), initialOrder, updated)
				}
			}
		})
	}
}
