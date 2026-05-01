package services

import "github.com/google/uuid"

// AccessTokenClaims is the JWT payload of an access token issued by this service. It is
// embedded in the JWT signed by the json-keys service and decoded back by the auth
// middleware on every authenticated request.
//
// Anonymous tokens (issued by tokenCreateAnon) leave UserID nil and Roles empty; those
// tokens grant access only to endpoints that explicitly opt into optional auth.
type AccessTokenClaims struct {
	// UserID is the authenticated user's UUID. Nil on anonymous access tokens.
	UserID *uuid.UUID `json:"userID,omitempty"`
	// Roles are the role names granted to this user at sign time. Empty for anonymous tokens.
	// Permissions are resolved from the role names by the auth middleware at request time,
	// so role rename / permission change takes effect on the next request without re-issuing.
	Roles []string `json:"roles,omitempty"`
	// RefreshTokenID is the JTI of the refresh token that minted this access token. The
	// token-refresh flow checks that the refresh token's own JTI matches this field, which
	// binds the access/refresh pair: revoking a refresh token effectively revokes every
	// access token derived from it.
	RefreshTokenID string `json:"refreshTokenID,omitempty"`
}
