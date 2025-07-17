package subscriber

import (
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/abgdnv/gocommerce/pkg/messaging/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type mockAckableMsg struct {
	mock.Mock
}

func (m *mockAckableMsg) Data() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *mockAckableMsg) Ack() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockAckableMsg) Nak() error {
	args := m.Called()
	return args.Error(0)
}

func Test_handleMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	testCases := []struct {
		name       string
		newMockMsg func() *mockAckableMsg
	}{
		{
			name: "valid message",
			newMockMsg: func() *mockAckableMsg {
				validPayload, _ := json.Marshal(&events.OrderCreatedEvent{
					OrderID:    uuid.New(),
					UserID:     uuid.New(),
					TotalPrice: 1000,
					CreatedAt:  time.Now(),
				})
				msg := new(mockAckableMsg)
				msg.On("Data").Return(validPayload).Times(1)
				msg.On("Ack").Return(nil).Times(1)
				return msg
			},
		},
		{
			name: "invalid message",
			newMockMsg: func() *mockAckableMsg {
				msg := new(mockAckableMsg)
				msg.On("Data").Return([]byte("invalid data")).Times(1)
				msg.On("Nak").Return(nil).Times(1)
				return msg
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			mockMsg := tc.newMockMsg()

			// when
			handleMessage(mockMsg, logger)

			// then
			mockMsg.AssertExpectations(t)
		})
	}
}
