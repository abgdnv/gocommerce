package rest

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateReverseProxyWithRewrite(t *testing.T) {
	type proxyConfig struct {
		targetURL string
		fromPath  string
		toPath    string
	}

	testCases := []struct {
		name                string
		cfg                 proxyConfig
		incomingURL         string
		expectErr           bool
		expectedBackendPath string
	}{
		{
			name: "Success - should rewrite path correctly",
			cfg: proxyConfig{
				targetURL: "", //will be replaced with test server URL
				fromPath:  "/api/products",
				toPath:    "/internal/v1/products",
			},
			incomingURL:         "http://gateway/api/products/123?q=test",
			expectErr:           false,
			expectedBackendPath: "/internal/v1/products/123?q=test",
		},
		{
			name: "Success - root path",
			cfg: proxyConfig{
				targetURL: "",
				fromPath:  "/",
				toPath:    "/api/v1/",
			},
			incomingURL:         "http://gateway/some/path",
			expectErr:           false,
			expectedBackendPath: "/api/v1/some/path",
		},
		{
			name: "Error - invalid target URL",
			cfg: proxyConfig{
				targetURL: "://invalid-url",
				fromPath:  "/from",
				toPath:    "/to",
			},
			incomingURL: "http://gateway/from/123",
			expectErr:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// given

			var receivedPath string
			var wg sync.WaitGroup
			wg.Add(1)

			// 1. Creating test server
			backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.RequestURI() // saving a path
				wg.Done()
				w.WriteHeader(http.StatusOK)
			}))
			defer backendServer.Close()

			// 2. This is the target URL for the proxy
			if tc.cfg.targetURL == "" {
				tc.cfg.targetURL = backendServer.URL
			}

			// when
			proxyHandler, err := createReverseProxyWithRewrite(tc.cfg.targetURL, tc.cfg.fromPath, tc.cfg.toPath)
			// then
			if tc.expectErr {
				require.Error(t, err, "Expected an error during proxy creation, but got none")
				require.Nil(t, proxyHandler, "Handler should be nil on error")
				return // Тест завершен, если мы ожидали ошибку
			}
			require.NoError(t, err, "Proxy creation failed")
			require.NotNil(t, proxyHandler, "Handler should not be nil")

			// when
			req := httptest.NewRequest(http.MethodGet, tc.incomingURL, nil)
			rr := httptest.NewRecorder()
			proxyHandler.ServeHTTP(rr, req)
			wg.Wait()

			// then
			assert.Equal(t, tc.expectedBackendPath, receivedPath, "The backend server received a request with an unexpected path")
			assert.Equal(t, http.StatusOK, rr.Code, "The proxy handler should return the status code from the backend")
		})
	}
}
