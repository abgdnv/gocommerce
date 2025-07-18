package subscriber

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/messaging/events"
	pnats "github.com/abgdnv/gocommerce/pkg/nats"
	"github.com/google/uuid"
	natsgo "github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/nats"
	"golang.org/x/sync/errgroup"
)

// skipIntegrationTests is the environment variable that controls whether to skip integration tests.
const skipIntegrationTests = "NOTIFICATION_SVC_SKIP_INTEGRATION_TESTS"
const natsImg = "nats:2.11.6-alpine"

// SubscriberSuite is a test suite for testing the NATS subscriber functionality.
type SubscriberSuite struct {
	suite.Suite                           // Embedding testify suite for structured testing
	ctx           context.Context         // Context for the test suite, used for cancellation and timeouts
	logger        *slog.Logger            // Logger for the test suite
	natsContainer *nats.NATSContainer     // NATS container for running tests
	jsCtx         natsgo.JetStreamContext // JetStream context for NATS operations
	nc            *natsgo.Conn            // NATS connection for the subscriber
}

// SetupSuite initializes the test suite, setting up the NATS container and JetStream context.
func (s *SubscriberSuite) SetupSuite() {
	s.ctx = context.Background()
	s.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	var err error

	s.natsContainer, err = nats.Run(s.ctx, natsImg)
	require.NoError(s.T(), err, "Failed to run NATS container")

	natsURL, _ := s.natsContainer.ConnectionString(s.ctx)
	s.nc, err = natsgo.Connect(natsURL)
	require.NoError(s.T(), err, "Failed to connect to NATS")

	s.jsCtx, err = s.nc.JetStream()
	require.NoError(s.T(), err, "Failed to get JetStream context")

	s.logger.Info("Initialization complete for SubscribeSuite")
}

// TearDownSuite cleans up the NATS container after tests are done.
func (s *SubscriberSuite) TearDownSuite() {
	s.logger.Info("Terminating NATS container...")
	s.nc.Close() // Close the NATS connection
	err := testcontainers.TerminateContainer(s.natsContainer)
	if err != nil {
		s.logger.Error("Failed to terminate NATS container", "error", err)
		return
	}
	s.logger.Info("NATS container terminated successfully.")
}

// TestSubscriberIntegration runs the test suite for the NATS subscriber integration tests.
func TestSubscriberIntegration(t *testing.T) {
	if os.Getenv(skipIntegrationTests) == "1" {
		t.Skip("Skipping integration tests based on " + skipIntegrationTests + " env var")
	}
	// Run the test suite
	suite.Run(t, new(SubscriberSuite))
}

// TestCaseConfig defines the configuration for each test case in the subscriber tests.
type TestCaseConfig struct {
	name         string
	streamName   string
	consumerName string
	subjectName  string
	publish      func(js natsgo.JetStreamContext, testSubject string) error
	condition    func(testStream string, testConsumer string) bool
	assert       func(testStream string, testConsumer string)
}

// TestReceiveMessage tests the message receiving functionality of the NATS subscriber.
func (s *SubscriberSuite) TestReceiveMessage() {
	// given
	testCases := []TestCaseConfig{
		{
			name:         "Successfully receive message",
			streamName:   "STREAM-" + uuid.NewString(),
			consumerName: "CONSUMER-" + uuid.NewString(),
			subjectName:  "subject." + uuid.NewString(),
			publish: func(js natsgo.JetStreamContext, testSubject string) error {
				testEvent := events.OrderCreatedEvent{
					OrderID:    uuid.New(),
					UserID:     uuid.New(),
					TotalPrice: 9999,
					CreatedAt:  time.Now(),
				}
				payload, _ := testEvent.Payload()
				testMessage := &natsgo.Msg{
					Subject: testSubject,
					Data:    payload,
				}
				_, err := js.PublishMsg(testMessage)
				return err
			},
			condition: func(testStream, testConsumer string) bool {
				consumerInfo, err := s.jsCtx.ConsumerInfo(testStream, testConsumer)
				if err != nil {
					return false
				}
				return consumerInfo.NumAckPending == 0 && consumerInfo.NumPending == 0
			},
			assert: func(testStream, testConsumer string) {
				finalConsumerInfo, err := s.jsCtx.ConsumerInfo(testStream, testConsumer)
				require.NoError(s.T(), err)
				// Assert that the consumer has no messages pending acknowledgment
				require.Zero(s.T(), finalConsumerInfo.NumAckPending)
				// Assert that the consumer has no messages in the queue
				require.Zero(s.T(), finalConsumerInfo.NumPending)
			},
		},
		{
			name:         "Invalid payload",
			streamName:   "STREAM_" + uuid.NewString(),
			consumerName: "CONSUMER_" + uuid.NewString(),
			subjectName:  "subject." + uuid.NewString(),
			publish: func(js natsgo.JetStreamContext, testSubject string) error {
				// Publish an invalid message that cannot be unmarshalled
				invalidMessage := &natsgo.Msg{
					Subject: testSubject,
					Data:    []byte("invalid payload"),
				}
				_, err := js.PublishMsg(invalidMessage)
				if err != nil {
					return err
				}

				// Publish valid message to ensure the subscriber is still running
				validEvent := events.OrderCreatedEvent{
					OrderID:    uuid.New(),
					UserID:     uuid.New(),
					TotalPrice: 9999,
					CreatedAt:  time.Now(),
				}
				payload, _ := validEvent.Payload()
				validMessage := &natsgo.Msg{
					Subject: testSubject,
					Data:    payload,
				}
				_, err = js.PublishMsg(validMessage)
				return err
			},
			condition: func(testStream, testConsumer string) bool {
				consumerInfo, err := s.jsCtx.ConsumerInfo(testStream, testConsumer)
				if err != nil {
					return false
				}
				return consumerInfo.NumPending == uint64(0) && consumerInfo.AckFloor.Stream == uint64(2)
			},
			assert: func(testStream, testConsumer string) {
				finalConsumerInfo, err := s.jsCtx.ConsumerInfo(testStream, testConsumer)
				require.NoError(s.T(), err)
				// Assert that the consumer has no pending messages
				require.Equal(s.T(), uint64(0), finalConsumerInfo.NumPending)
				// Assert that the stream's AckFloor is set to 2, indicating the invalid message was processed
				require.Equal(s.T(), uint64(2), finalConsumerInfo.AckFloor.Stream)
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			s.runTest(t, &tc)
		})
	}
}

// runTest executes a single test case for the NATS subscriber.
func (s *SubscriberSuite) runTest(t *testing.T, tc *TestCaseConfig) {
	// Set up a test context with a timeout to ensure the test does not hang indefinitely
	testCtx, testCancel := context.WithTimeout(s.ctx, 6*time.Second)
	g, gCtx := errgroup.WithContext(testCtx)
	// Ensure the test is cleaned up properly
	t.Cleanup(func() {
		s.logger.Info("Cleaning up test resources...", slog.String("test_name", tc.name))
		testCancel()
		err := g.Wait()
		require.ErrorIs(s.T(), err, context.Canceled, "error should be context.Canceled")
	})
	// Create a new JetStream stream for the test
	_, err := s.jsCtx.AddStream(&natsgo.StreamConfig{
		Name:      tc.streamName,
		Subjects:  []string{tc.subjectName},
		Retention: natsgo.WorkQueuePolicy,
	})
	require.NoError(s.T(), err, "Failed to add stream to JetStream")

	// Initialize the subscriber with the configuration
	cfgSubscriber := config.SubscriberConfig{
		Stream:   tc.streamName,
		Subject:  tc.subjectName,
		Consumer: tc.consumerName,
		Timeout:  200 * time.Millisecond,
		Interval: 200 * time.Microsecond,
		Workers:  1,
	}
	js, err := pnats.NewJetStreamContext(s.nc)
	require.NoError(s.T(), err, "Failed to create JetStream context")
	g.Go(func() error {
		s.logger.Info("NATS subscriber started")
		return Start(gCtx, js, cfgSubscriber, s.logger)
	})

	// when
	err = tc.publish(s.jsCtx, tc.subjectName)
	require.NoError(s.T(), err, "Failed to publish test message")

	// then
	require.Eventually(s.T(), func() bool {
		return tc.condition(tc.streamName, tc.consumerName)
	}, 5*time.Second, 100*time.Millisecond, "No messages received within the timeout period")

	// Assert the final state of the consumer
	tc.assert(tc.streamName, tc.consumerName)

}
