package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/abgdnv/gocommerce/pkg/auth"
)

type contextKey string

const UserIDContextKey = contextKey("userID")

// AuthMiddleware is a middleware that verifies JWT tokens in the Authorization header.
// It extracts the user ID from the token and adds it to the request context.
// If the token is invalid or missing, it returns a 401 Unauthorized response.
// If the token is valid, it calls the next handler in the chain.
// The user ID can be accessed in the next handlers via the context.
func AuthMiddleware(verifier auth.Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader { // Если префикса не было
				http.Error(w, "Bearer token is required", http.StatusUnauthorized)
				return
			}

			token, err := verifier.Verify(r.Context(), tokenString)
			if err != nil {
				http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// get the user ID from the token claims
			subject, ok := token.Subject()
			if !ok {
				http.Error(w, "no claim `sub`", http.StatusUnauthorized)
				return
			}
			// Enrich the request context with the user ID.
			ctx := context.WithValue(r.Context(), UserIDContextKey, subject)

			// Pass the enriched context to the next handler in the chain.
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ContextUserID retrieves the user ID from the context.
func ContextUserID(ctx context.Context) string {
	value := ctx.Value(UserIDContextKey)
	if value != nil {
		return value.(string)
	}
	return ""
}
