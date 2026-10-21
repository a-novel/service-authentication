package session

import (
	"context"
	"errors"
	"fmt"
)

var (
	// ErrMissingAuth is returned by [MustGetClaimsContext] when the context carries no claims.
	ErrMissingAuth = errors.New("missing auth")
	// ErrUnexpectedClaims is returned when the value stored under [ClaimsContextKey] is not a
	// [*Claims].
	ErrUnexpectedClaims = errors.New("unexpected claims")
)

// ClaimsContextKey is the context key under which the auth middleware stores the verified claims.
type ClaimsContextKey struct{}

// SetClaimsContext returns a copy of ctx carrying claims. The auth middleware calls it once it has
// verified a token.
func SetClaimsContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey{}, claims)
}

// GetClaimsContext returns the claims ctx carries. It returns nil claims and no error when the
// request carried no token, as on an optional-auth endpoint. It returns [ErrUnexpectedClaims] when
// the value under [ClaimsContextKey] is not a [*Claims].
func GetClaimsContext(ctx context.Context) (*Claims, error) {
	raw := ctx.Value(ClaimsContextKey{})
	if raw == nil {
		return nil, nil
	}

	claims, ok := raw.(*Claims)
	if !ok {
		return nil, fmt.Errorf("%w: got type %T, expected %T", ErrUnexpectedClaims, raw, claims)
	}

	return claims, nil
}

// MustGetClaimsContext returns the claims ctx carries, and [ErrMissingAuth] when it carries none.
// Use it on endpoints that require authentication, and [GetClaimsContext] where authentication is
// optional.
func MustGetClaimsContext(ctx context.Context) (*Claims, error) {
	claims, err := GetClaimsContext(ctx)
	if err != nil {
		return nil, err
	}

	if claims == nil {
		return nil, ErrMissingAuth
	}

	return claims, nil
}
