package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

type Credentials struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func loadCredentials(s *core.Credentials) Credentials {
	return Credentials{
		ID:        s.ID,
		Email:     s.Email,
		Role:      s.Role,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func loadCredentialsMap(item *core.Credentials, _ int) Credentials {
	return loadCredentials(item)
}
