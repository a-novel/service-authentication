package models

import "github.com/google/uuid"

// AccessTokenClaims is authenticated (signed) information about a session. This data is embed in the
// access tokens.
type AccessTokenClaims struct {
	// ID of the user that created the session. Can be empty for anonymous sessions.
	UserID *uuid.UUID `json:"userID,omitempty"`
	// Roles is a list of Role attributed to the session.
	Roles []Role `json:"roles"`
}
