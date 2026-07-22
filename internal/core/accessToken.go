package core

import "github.com/google/uuid"

// AccessTokenClaims is the JWT payload of an access token issued by this service. It is
// embedded in the JWT signed by the json-keys service and decoded back by the auth
// middleware on every authenticated request.
//
// Anonymous tokens (issued by tokenCreateAnon) leave UserID nil and set Roles to the single
// anonymous role; those tokens grant access only to endpoints that explicitly opt into
// optional auth.
type AccessTokenClaims struct {
	// UserID is the authenticated user's UUID. Nil on anonymous access tokens.
	UserID *uuid.UUID `json:"userID,omitempty"`
	// Roles are the role names granted to this user at sign time. The auth middleware
	// resolves them to permissions at request time, so a permission added to a role takes
	// effect on the next request; renaming or removing a role requires re-issuing every
	// token that still names it.
	Roles []string `json:"roles,omitempty"`
	// RefreshTokenID is the JTI of the refresh token that minted this access token. The
	// token-refresh flow requires the two to match, which binds the pair: revoking a
	// refresh token revokes every access token derived from it.
	RefreshTokenID string `json:"refreshTokenID,omitempty"`
}
