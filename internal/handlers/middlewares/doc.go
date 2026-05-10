// Package middlewares holds the HTTP middleware stack for the authentication
// service. The [Auth] middleware verifies the Bearer access token via the
// json-keys claims verifier, resolves the caller's role-granted permissions
// against the configured permission map, and stores the verified claims on the
// request context for downstream handlers to retrieve via [GetClaimsContext].
package middlewares
