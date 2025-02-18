package models

import "github.com/google/uuid"

// AccessTokenClaims is authenticated (signed) information about a session. This data is embed in the
// access tokens.
type AccessTokenClaims struct {
	// ID of the user that created the session. Can be empty for anonymous sessions.
	UserID *uuid.UUID `json:"userID,omitempty"`
	// Roles is a list of Role attributed to the session.
	Roles []Role `json:"roles"`
	// The ID of the refresh token that created the session. If the session was created using direct login, this
	// field is empty.
	RefreshTokenID *string `json:"refreshTokenID,omitempty"`
}

type RefreshTokenClaims struct {
	// ID of the user that created the session. Can be empty for anonymous sessions.
	Jti string `json:"jti"`
	// ID of the user that created the session.
	UserID uuid.UUID `json:"userID,omitempty"`
}
