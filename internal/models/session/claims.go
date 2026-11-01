package session

import (
	"cmp"

	"github.com/google/uuid"
)

// Claims is the JWT payload of an access token issued by this service. JSON Keys signs it, and the
// auth middleware decodes it on every authenticated request.
//
// An anonymous token leaves UserID nil and carries the anonymous role alone. It reaches the routes
// that role's permissions open.
type Claims struct {
	// UserID is the authenticated user's UUID. Nil on anonymous access tokens.
	UserID *uuid.UUID `json:"userID,omitempty"`
	// Roles are the role names granted to this user at sign time. The auth middleware maps
	// them to permissions on each request, and fails a role the configuration does not define
	// with a server error.
	Roles []string `json:"roles,omitempty"`
	// RefreshTokenID is the JTI of the refresh token that minted this access token. Only
	// that refresh token renews it.
	RefreshTokenID string `json:"refreshTokenID,omitempty"`
	// SessionID is the session the token belongs to, read through [Claims.Session].
	SessionID string `json:"sessionID,omitempty"`
	// SessionExp is when the session ends, as a Unix timestamp in seconds: the expiry of the refresh
	// token that opened it. Zero on anonymous tokens.
	SessionExp int64 `json:"sessionExp,omitempty"`
}

// Session returns the ID of the session the token belongs to, and an empty string for an anonymous
// token. A token without a SessionID belongs to the session its refresh token opened.
func (claims *Claims) Session() string {
	return cmp.Or(claims.SessionID, claims.RefreshTokenID)
}

// RefreshClaims is the JWT payload of a refresh token: the token's own JTI, its expiry and the user
// it authenticates. Renewing a session matches Jti against the access token's [Claims.RefreshTokenID].
type RefreshClaims struct {
	Jti string `json:"jti,omitempty"`
	// Exp is the token's expiry, as a Unix timestamp in seconds. The signer sets it.
	Exp    int64     `json:"exp,omitempty"`
	UserID uuid.UUID `json:"userID,omitempty"`
	// SessionID is the session the token belongs to, read through [RefreshClaims.Session].
	SessionID string `json:"sessionID,omitempty"`
	// SessionExp is when the session ends, read through [RefreshClaims.SessionExpiry].
	SessionExp int64 `json:"sessionExp,omitempty"`
	// ClientID is the OAuth client the token was issued to, empty for a token the account platform
	// holds. Only that client refreshes it, at the token endpoint (RFC 6749 §10.4).
	ClientID string `json:"clientID,omitempty"`
}

// Session returns the ID of the session the token belongs to. A token without a SessionID, such as
// a password sign-in's, opened its own session, so its Jti is the session's ID.
func (claims *RefreshClaims) Session() string {
	return cmp.Or(claims.SessionID, claims.Jti)
}

// SessionExpiry returns when the session the token belongs to ends, as a Unix timestamp in seconds.
// A token that opened its own session ends it when it expires.
func (claims *RefreshClaims) SessionExpiry() int64 {
	return cmp.Or(claims.SessionExp, claims.Exp)
}

// RefreshClaimsForm is the payload submitted to the signer to mint a refresh token. The signer
// assigns the JTI, so only the user ID and the session are supplied here.
type RefreshClaimsForm struct {
	UserID uuid.UUID `json:"userID,omitempty"`
	// SessionID joins the token to an existing session. Empty, the token opens a session of its own.
	SessionID string `json:"sessionID,omitempty"`
	// SessionExp is when the joined session ends, as a Unix timestamp in seconds. The token stops
	// renewing then, whatever its own expiry.
	SessionExp int64 `json:"sessionExp,omitempty"`
	// ClientID binds the token to the OAuth client it is issued to.
	ClientID string `json:"clientID,omitempty"`
}
