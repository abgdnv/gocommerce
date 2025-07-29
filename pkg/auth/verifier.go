package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

type Verifier interface {
	Verify(ctx context.Context, tokenString string) (jwt.Token, error)
}

// JWTVerifier manages JWT verification using a JWKS endpoint.
// It caches the JWKS set to minimize network calls and supports automatic refresh.
type JWTVerifier struct {
	mu sync.RWMutex

	jwksURL  string
	issuer   string
	clientID string

	cachedSet     jwk.Set
	lastRefreshed time.Time
	minInterval   time.Duration
}

// NewJWTVerifier creates a new JWTVerifier instance.
func NewJWTVerifier(ctx context.Context, cfg config.IdP) (*JWTVerifier, error) {
	v := &JWTVerifier{
		jwksURL:     cfg.JwksURL,
		issuer:      cfg.Issuer,
		clientID:    cfg.ClientID,
		minInterval: cfg.MinInterval,
	}
	// Fail-Fast: Immediately fetch the JWKS to ensure the configuration is valid.
	if _, err := v.getKeySet(ctx); err != nil {
		return nil, fmt.Errorf("initial JWKS fetch failed: %w", err)
	}

	return v, nil
}

// getKeySet retrieves the JWKS set, caching it for subsequent calls.
func (v *JWTVerifier) getKeySet(ctx context.Context) (jwk.Set, error) {
	// Check if the cached set is still valid
	// If it is, return it immediately to avoid unnecessary network calls.
	v.mu.RLock()
	if v.cachedSet != nil && time.Since(v.lastRefreshed) < v.minInterval {
		set := v.cachedSet
		v.mu.RUnlock()
		return set, nil
	}
	v.mu.RUnlock()

	// If the cached set is not valid or not present, fetch it from the JWKS URL.
	// Use a write lock to ensure thread safety while updating the cache.
	// This prevents multiple goroutines from fetching the JWKS simultaneously.
	v.mu.Lock()
	defer v.mu.Unlock()
	// Double-check if the cached set is still valid after acquiring the lock.
	// This prevents unnecessary network calls if another goroutine has already updated the cache.
	if v.cachedSet != nil && time.Since(v.lastRefreshed) < v.minInterval {
		return v.cachedSet, nil
	}
	set, err := jwk.Fetch(ctx, v.jwksURL)
	if err != nil {
		// If the fetch fails, return the cached set if available, or an error.
		// This ensures that the application can still function if the JWKS endpoint is temporarily unavailable.
		if v.cachedSet != nil {
			return v.cachedSet, nil
		}
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", v.jwksURL, err)
	}
	v.cachedSet = set
	v.lastRefreshed = time.Now()
	return v.cachedSet, nil
}

func (v *JWTVerifier) Verify(ctx context.Context, tokenString string) (jwt.Token, error) {
	set, err := v.getKeySet(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get keyset for verification: %w", err)
	}

	token, err := jwt.Parse(
		[]byte(tokenString),
		jwt.WithKeySet(set),
		// Standard validation checks - expiration, not before, etc.
		jwt.WithValidate(true),
		// Validate the issuer
		jwt.WithIssuer(v.issuer),
		// Validate the authorized party (client ID)
		jwt.WithClaimValue("azp", v.clientID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}
	return token, nil
}
