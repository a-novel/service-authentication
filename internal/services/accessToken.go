package services

import "github.com/google/uuid"

type AccessTokenClaims struct {
	UserID         *uuid.UUID `json:"userID,omitempty"`
	Roles          []string   `json:"roles,omitempty"`
	RefreshTokenID string     `json:"refreshTokenID,omitempty"`
}
