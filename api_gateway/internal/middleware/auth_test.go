package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockVerifier is a mock implementation of the auth.Verifier interface for testing purposes.
type MockVerifier struct {
	mock.Mock
}

func (m *MockVerifier) Verify(ctx context.Context, tokenString string) (jwt.Token, error) {
	args := m.Called(ctx, tokenString)

	var token jwt.Token
	if args.Get(0) != nil {
		token = args.Get(0).(jwt.Token)
	}
	return token, args.Error(1)
}

func TestAuthMiddleware(t *testing.T) {
	// given

	// Create a mock of a valid JWT token
	mockValidToken, err := jwt.NewBuilder().
		Subject("user-123").
		Issuer("test-issuer").
		Audience([]string{"test-client"}).
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(time.Hour)).
		Build()
	require.NoError(t, err)

	testCases := []struct {
		name               string
		authHeader         string                // Authorization header to simulate the request
		setupMock          func(m *MockVerifier) // Function to set up our mock
		expectedStatusCode int
		shouldCallNext     bool   // Whether the next handler should be called
		expectedUserID     string // userID expected in the context
	}{
		{
			name:       "Success - valid bearer token",
			authHeader: "Bearer valid-token",
			setupMock: func(m *MockVerifier) {
				m.On("Verify", mock.Anything, "valid-token").Return(mockValidToken, nil)
			},
			expectedStatusCode: http.StatusOK,
			shouldCallNext:     true,
			expectedUserID:     "user-123",
		},
		{
			name:       "Failure - no auth header",
			authHeader: "",
			setupMock: func(m *MockVerifier) { // Nothing to set up, Verify should not be called
			},
			expectedStatusCode: http.StatusUnauthorized,
			shouldCallNext:     false,
		},
		{
			name:       "Failure - not a bearer token",
			authHeader: "Basic some-credentials",
			setupMock: func(m *MockVerifier) { // Nothing to set up, Verify should not be called
			},
			expectedStatusCode: http.StatusUnauthorized,
			shouldCallNext:     false,
		},
		{
			name:       "Failure - verifier returns error",
			authHeader: "Bearer invalid-token",
			setupMock: func(m *MockVerifier) {
				// Simulate an error from the verifier
				m.On("Verify", mock.Anything, "invalid-token").Return(nil, errors.New("signature is invalid"))
			},
			expectedStatusCode: http.StatusUnauthorized,
			shouldCallNext:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockVerifier := new(MockVerifier)
			tc.setupMock(mockVerifier)
			// Create the auth middleware with the mock verifier
			authMiddleware := AuthMiddleware(mockVerifier)

			// nextHandlerCalled - a flag to check if the next handler was called
			nextHandlerCalled := false
			// This is the next handler that should be called if the auth middleware passes
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextHandlerCalled = true
				// Check if the userID is in the context
				userID, ok := r.Context().Value(UserIDContextKey).(string)
				assert.True(t, ok, "userID should be in context")
				assert.Equal(t, tc.expectedUserID, userID, "userID in context is incorrect")
				w.WriteHeader(http.StatusOK)
			})

			// Create the test handler with the auth middleware that wraps the next handler
			// If the auth middleware fails, this handler should not be called
			// and the status code should be 401 Unauthorized
			testHandler := authMiddleware(nextHandler)

			// create a request with the auth header if provided
			req := httptest.NewRequest("GET", "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rr := httptest.NewRecorder()

			// when
			testHandler.ServeHTTP(rr, req)

			// then
			assert.Equal(t, tc.expectedStatusCode, rr.Code, "HTTP status code is wrong")
			assert.Equal(t, tc.shouldCallNext, nextHandlerCalled, "Next handler call status is wrong")

			// Check if all expected calls on the mock were made
			mockVerifier.AssertExpectations(t)
		})
	}
}
