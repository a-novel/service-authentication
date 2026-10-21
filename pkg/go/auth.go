// Package serviceauthentication is the Go client for the authentication service.
//
// The client mounts the auth middleware on a JSON Keys claims verifier. A service builds a
// [PermissionsHandler] with [NewAuthHandler] at startup, then gates each route on the
// permissions it requires. Helpers read the authenticated user's claims from the request
// context.
package serviceauthentication

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/logging"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// Role is a named bundle of permissions assigned to a user.
type Role = authconfig.Role

// Permissions is the configured role-permission map for a deployment.
type Permissions = authconfig.Permissions

// Claims is the authenticated user's access-token payload, extracted from the JWT and
// stored in the request context by the auth middleware.
type Claims = session.Claims

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
	permissionsByRole := lo.Must(lib.ResolveDependants(
		lo.MapEntries(permissions.Roles, rolePermissions),
		lo.MapEntries(permissions.Roles, roleInherits),
	))

	middlewareAuth := middlewares.NewAuth(claimsVerifier, permissionsByRole, logger)

	return func(r chi.Router, permissions ...string) chi.Router {
		return r.With(middlewareAuth.Middleware(permissions))
	}
}

func rolePermissions(name string, role authconfig.Role) (string, []string) {
	return name, role.Permissions
}

func roleInherits(name string, role authconfig.Role) (string, []string) {
	return name, role.Inherits
}

// SetClaimsContext stores the authenticated user's claims in the context. The auth
// middleware calls this after successful token verification; downstream handlers should
// not need to call it directly.
func SetClaimsContext(ctx context.Context, claims *Claims) context.Context {
	return session.SetClaimsContext(ctx, claims)
}

// CallerKey keys a rate limit by the signed-in user, else by the visitor IP, for golib's
// httpf.Limits.By. Mount the limit after the auth middleware, which signs the user in.
func CallerKey(r *http.Request) (string, error) {
	return middlewares.CallerKey(r)
}

// GetClaimsContext retrieves the authenticated user's claims from the context, if any.
// Returns (nil, nil) for unauthenticated requests on optional-auth endpoints, and an error
// if the value stored under the claims key has an unexpected type.
func GetClaimsContext(ctx context.Context) (*Claims, error) {
	return session.GetClaimsContext(ctx)
}

// MustGetClaimsContext retrieves the authenticated user's claims from the context, returning
// an error if no claims are present. Use this on endpoints that require authentication; use
// [GetClaimsContext] on endpoints where authentication is optional.
func MustGetClaimsContext(ctx context.Context) (*Claims, error) {
	return session.MustGetClaimsContext(ctx)
}
