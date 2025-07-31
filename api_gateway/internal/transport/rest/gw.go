package rest

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	sCfg "github.com/abgdnv/gocommerce/api_gateway/internal/config"
	"github.com/abgdnv/gocommerce/api_gateway/internal/middleware"
	"github.com/abgdnv/gocommerce/api_gateway/internal/service"
	"github.com/abgdnv/gocommerce/pkg/auth"
	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/server"
	"github.com/abgdnv/gocommerce/pkg/web"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GW struct {
	httpCfg     config.HTTPConfig
	cfg         sCfg.Services
	userService *service.UserService
	logger      *slog.Logger
}

func NewGW(httpCfg config.HTTPConfig, userService *service.UserService, cfg sCfg.Services, logger *slog.Logger) *GW {
	return &GW{
		httpCfg:     httpCfg,
		cfg:         cfg,
		userService: userService,
		logger:      logger.With("component", "gw"),
	}
}

// SetupHTTPServer initializes the HTTP server with the configured reverse proxies.
// If there is an error creating the reverse proxy, it returns an error.
func (gw *GW) SetupHTTPServer(verifier *auth.JWTVerifier) (*http.Server, error) {
	mux := server.NewChiRouter(gw.logger)

	productProxy, err := createReverseProxyWithRewrite(gw.cfg.Product.Url, gw.cfg.Product.From, gw.cfg.Product.To)
	if err != nil {
		return nil, fmt.Errorf("failed to create product proxy: %w", err)
	}
	mux.Route(gw.cfg.Product.From, func(r chi.Router) {
		r.With(middleware.AuthMiddleware(verifier)).Post("/", productProxy.ServeHTTP)
		r.With(middleware.AuthMiddleware(verifier)).Put("/{id}", productProxy.ServeHTTP)
		r.With(middleware.AuthMiddleware(verifier)).Delete("/{id}", productProxy.ServeHTTP)

		r.With(middleware.AuthMiddleware(verifier)).Put("/{id}/stock", productProxy.ServeHTTP)

		r.Get("/", productProxy.ServeHTTP)
		r.Get("/{id}", productProxy.ServeHTTP)
	})

	mux.Group(func(r chi.Router) {
		r.Post(gw.cfg.User.From, gw.userRegisterHandler())
	})

	orderProxy, err := createReverseProxyWithRewrite(gw.cfg.Order.Url, gw.cfg.Order.From, gw.cfg.Order.To)
	if err != nil {
		return nil, fmt.Errorf("failed to create order proxy: %w", err)
	}
	mux.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(verifier))
		r.Mount(gw.cfg.Order.From, orderProxy)
	})

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
		userID := middleware.ContextUserID(req.Context())
		if userID != "" {
			req.Header.Set(web.XUserId, userID)
		}
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = toPath + strings.TrimPrefix(req.URL.Path, fromPath)
	}
	return proxy, nil
}

func (gw *GW) userRegisterHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mLogger := gw.loggerWithReqID(r)
		var userDto service.UserDto
		if err := json.NewDecoder(r.Body).Decode(&userDto); err != nil {
			mLogger.ErrorContext(r.Context(), "Error decoding request body", "error", err)
			web.RespondError(w, mLogger, http.StatusBadRequest, "Invalid request body")
			return
		}
		mLogger.DebugContext(r.Context(), "Received request to register user", "user", userDto)
		userID, err := gw.userService.Register(r.Context(), userDto)
		if err != nil {
			s, ok := status.FromError(err)
			var httpStatus int
			if ok {
				switch s.Code() {
				case codes.AlreadyExists:
					httpStatus = http.StatusConflict
				case codes.InvalidArgument:
					httpStatus = http.StatusBadRequest
				default:
					httpStatus = http.StatusInternalServerError
				}
				web.RespondError(w, mLogger, httpStatus, s.Message())
				return
			}
			web.RespondError(w, mLogger, http.StatusInternalServerError, "User registration error")
			return
		}
		web.RespondJSON(w, mLogger, http.StatusCreated, map[string]string{"id": *userID})
	}
}

// loggerWithReqID creates a logger with the request ID from the context.
func (gw *GW) loggerWithReqID(r *http.Request) *slog.Logger {
	reqID, found := web.GetRequestID(r.Context())
	if !found {
		reqID = "unknown"
	}
	return gw.logger.With("request_id", reqID)
}
