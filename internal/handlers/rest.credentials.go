package handlers

import (
	"time"

	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// Credentials is the JSON representation of a user's account record returned by
// the credentials endpoints.
type Credentials struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CredentialsSelf is the JSON representation of the caller's own account, returned by
// GET /v2/credentials/self.
type CredentialsSelf struct {
	Credentials

	PendingEmail *PendingEmail `json:"pendingEmail,omitempty"`
}

// PendingEmail is the JSON representation of an email change awaiting confirmation.
type PendingEmail struct {
	Email string `json:"email"`

	ExpiresAt time.Time `json:"expiresAt"`
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

func loadCredentialsSelf(s *core.CredentialsSelf) CredentialsSelf {
	res := CredentialsSelf{Credentials: loadCredentials(&s.Credentials)}

	if s.PendingEmail != nil {
		res.PendingEmail = &PendingEmail{Email: s.PendingEmail.Email, ExpiresAt: s.PendingEmail.ExpiresAt}
	}

	return res
}

func loadCredentialsMap(item *core.Credentials, _ int) Credentials {
	return loadCredentials(item)
}
