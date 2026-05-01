// Package serviceauthentication is the Go client for the authentication service.
//
// The package wires the auth middleware against the JSON-keys claims verifier and exposes
// helpers for retrieving the authenticated user's claims from a request context. Consumers
// typically construct a [PermissionsHandler] with [NewAuthHandler] at startup and use it to
// gate individual routes by required permission.
package serviceauthentication

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/deps"
	"github.com/a-novel-kit/golib/logging"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

// Role is a named bundle of permissions assigned to a user.
type Role = config.Role

// Permissions is the configured role-permission map for a deployment.
type Permissions = config.Permissions

// Claims is the authenticated user's access-token payload, extracted from the JWT and
// stored in the request context by the auth middleware.
type Claims = services.AccessTokenClaims

// PermissionsHandler returns a chi sub-router that enforces the listed permissions for the
// routes mounted on it. Pass zero permissions for optional authentication: the request is
// allowed through without an Authorization header, and a valid bearer token (if present)
// still populates [Claims] in the request context for handlers that branch on identity.
type PermissionsHandler func(r chi.Router, permissions ...string) chi.Router

// NewAuthHandler constructs a [PermissionsHandler] backed by the given claims verifier and
// permission map. Role inheritance is resolved at startup: a role inherits every permission
// transitively granted by the roles in its Inherits list, so route mounts only need to
// reference leaf permissions.
func NewAuthHandler(
	claimsVerifier middlewares.AuthClaimsVerifier,
	permissions Permissions,
	logger logging.Log,
) PermissionsHandler {
	permissionsByRole := lo.Must(deps.ResolveDependants[string, string](
		lo.MapEntries(
			permissions.Roles,
			func(key string, value config.Role) (string, []string) {
				return key, value.Permissions
			},
		),
		lo.MapEntries(permissions.Roles, func(key string, value config.Role) (string, []string) {
			return key, value.Inherits
		}),
	))

	middlewareAuth := middlewares.NewAuth(claimsVerifier, permissionsByRole, logger)

	return func(r chi.Router, permissions ...string) chi.Router {
		return r.With(middlewareAuth.Middleware(permissions))
	}
}

// SetClaimsContext stores the authenticated user's claims in the context. The auth
// middleware calls this after successful token verification; downstream handlers should
// not need to call it directly.
func SetClaimsContext(ctx context.Context, claims *Claims) context.Context {
	return middlewares.SetClaimsContext(ctx, claims)
}

// GetClaimsContext retrieves the authenticated user's claims from the context, if any.
// Returns (nil, nil) for unauthenticated requests on optional-auth endpoints. Returns an
// error wrapping [middlewares.ErrUnexpectedClaims] if the value stored under the claims
// key has the wrong type.
func GetClaimsContext(ctx context.Context) (*Claims, error) {
	return middlewares.GetClaimsContext(ctx)
}

// MustGetClaimsContext retrieves the authenticated user's claims from the context, returning
// [middlewares.ErrMissingAuth] if no claims are present. Use this on endpoints that require
// authentication; use [GetClaimsContext] on endpoints where authentication is optional.
func MustGetClaimsContext(ctx context.Context) (*Claims, error) {
	return middlewares.MustGetClaimsContext(ctx)
}
