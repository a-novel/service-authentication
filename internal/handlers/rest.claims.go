package handlers

import "github.com/google/uuid"

// Claims is the JSON body of the claims endpoint: the identity a valid access
// token grants its bearer. UserID is nil for an anonymous token.
type Claims struct {
	UserID         *uuid.UUID `json:"userID,omitempty"`
	Roles          []string   `json:"roles,omitempty"`
	RefreshTokenID string     `json:"refreshTokenID,omitempty"`
	// SessionID is the session the token belongs to. Empty for an anonymous token.
	SessionID string `json:"sessionID,omitempty"`
	// SessionExp is when the session ends, as a Unix timestamp in seconds. Zero for an anonymous token.
	SessionExp int64 `json:"sessionExp,omitempty"`
	// ClientID is the OAuth client that holds the token, empty for the account platform's own tokens.
	ClientID string `json:"clientID,omitempty"`
}
