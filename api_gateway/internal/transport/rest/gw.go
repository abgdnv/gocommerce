package rest

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	sCfg "github.com/abgdnv/gocommerce/api_gateway/internal/config"
	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/server"
)

type GW struct {
	httpCfg config.HTTPConfig
	cfg     sCfg.Services
	logger  *slog.Logger
}

func NewGW(httpCfg config.HTTPConfig, cfg sCfg.Services, logger *slog.Logger) *GW {
	return &GW{
		cfg:     cfg,
		httpCfg: httpCfg,
		logger:  logger.With("component", "gw"),
	}
}

// SetupHTTPServer initializes the HTTP server with the configured reverse proxies.
// If there is an error creating the reverse proxy, it returns an error.
func (gw *GW) SetupHTTPServer() (*http.Server, error) {
	mux := server.NewChiRouter(gw.logger)

	productProxy, err := createReverseProxyWithRewrite(gw.cfg.Product.Url, gw.cfg.Product.From, gw.cfg.Product.To)
	if err != nil {
		return nil, fmt.Errorf("failed to create product proxy: %w", err)
	}
	mux.Mount(gw.cfg.Product.From, productProxy)

	orderProxy, err := createReverseProxyWithRewrite(gw.cfg.Order.Url, gw.cfg.Order.From, gw.cfg.Order.To)
	if err != nil {
		return nil, fmt.Errorf("failed to create order proxy: %w", err)
	}
	mux.Mount(gw.cfg.Order.From, orderProxy)

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", gw.httpCfg.Port),
		Handler:           mux,
		ReadTimeout:       gw.httpCfg.Timeout.Read,
		WriteTimeout:      gw.httpCfg.Timeout.Write,
		IdleTimeout:       gw.httpCfg.Timeout.Idle,
		ReadHeaderTimeout: gw.httpCfg.Timeout.ReadHeader,
		MaxHeaderBytes:    gw.httpCfg.MaxHeaderBytes,
	}, nil
}

// createReverseProxyWithRewrite creates a reverse proxy that rewrites the request path.
// It takes the target URL, the path to match, and the path to rewrite to.
// It returns an http.Handler that can be used in a router.
// If the target URL is invalid, it logs a fatal error and exits.
func createReverseProxyWithRewrite(targetURL, fromPath, toPath string) (http.Handler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL '%s': %w", targetURL, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	// Director will be called before the request is sent to the target.
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = strings.Replace(req.URL.Path, fromPath, toPath, 1)
	}
	return proxy, nil
}
