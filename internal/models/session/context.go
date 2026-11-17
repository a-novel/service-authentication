package session

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	// ErrMissingAuth is returned by [RequireClaimsContext] when the context carries no claims.
	ErrMissingAuth = errors.New("missing auth")
	// ErrAnonymousCaller is returned by [RequestUser] when the claims name no user, as an anonymous
	// session's do.
	ErrAnonymousCaller = errors.New("anonymous caller names no user")
)

// claimsContextKey is the context key under which [SetClaimsContext] stores the claims.
type claimsContextKey struct{}

// SetClaimsContext returns a copy of ctx carrying claims. The auth middleware calls it once it has
// verified a token.
func SetClaimsContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// GetClaimsContext returns the claims ctx carries, and nil when the request carried no token, as on
// an optional-auth endpoint.
func GetClaimsContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsContextKey{}).(*Claims)

	return claims
}

// RequireClaimsContext returns the claims ctx carries, and [ErrMissingAuth] when it carries none.
// Use it on endpoints that require authentication, and [GetClaimsContext] where authentication is
// optional.
func RequireClaimsContext(ctx context.Context) (*Claims, error) {
	claims := GetClaimsContext(ctx)
	if claims == nil {
		return nil, ErrMissingAuth
	}

	return claims, nil
}

// RequestUser returns the user an authenticated request acts for. It returns [ErrMissingAuth] when
// ctx carries no claims, and [ErrAnonymousCaller] when the claims name no user.
func RequestUser(ctx context.Context) (uuid.UUID, error) {
	claims, err := RequireClaimsContext(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	if claims.UserID == nil {
		return uuid.Nil, ErrAnonymousCaller
	}

	return *claims.UserID, nil
}
