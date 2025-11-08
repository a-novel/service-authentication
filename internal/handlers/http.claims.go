package handlers

import "github.com/google/uuid"

type Claims struct {
	UserID         *uuid.UUID `json:"userID,omitempty"`
	Roles          []string   `json:"roles,omitempty"`
	RefreshTokenID string     `json:"refreshTokenID,omitempty"`
}
