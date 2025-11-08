package services

import "github.com/google/uuid"

type RefreshTokenClaims struct {
	Jti    string    `json:"jti,omitempty"`
	UserID uuid.UUID `json:"userID,omitempty"`
}

type RefreshTokenClaimsForm struct {
	UserID uuid.UUID `json:"userID,omitempty"`
}
