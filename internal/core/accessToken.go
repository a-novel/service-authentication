package core

import "github.com/google/uuid"

// AccessTokenClaims is the JWT payload of an access token issued by this service. JSON Keys
// signs it, and the auth middleware decodes it on every authenticated request.
//
// An anonymous token, from [TokenCreateAnon], leaves UserID nil and carries the anonymous role
// alone. It reaches the routes that role's permissions open.
type AccessTokenClaims struct {
	// UserID is the authenticated user's UUID. Nil on anonymous access tokens.
	UserID *uuid.UUID `json:"userID,omitempty"`
	// Roles are the role names granted to this user at sign time. The auth middleware maps
	// them to permissions on each request, and fails a role the configuration does not define
	// with a server error.
	Roles []string `json:"roles,omitempty"`
	// RefreshTokenID is the JTI of the refresh token that minted this access token. Only
	// that refresh token renews it.
	RefreshTokenID string `json:"refreshTokenID,omitempty"`
}
