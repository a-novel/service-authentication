package handlers

import "github.com/google/uuid"

// Claims is the JSON body of the claims endpoint: the identity a valid access
// token grants its bearer. UserID is nil for an anonymous token.
type Claims struct {
	UserID         *uuid.UUID `json:"userID,omitempty"`
	Roles          []string   `json:"roles,omitempty"`
	RefreshTokenID string     `json:"refreshTokenID,omitempty"`
}
