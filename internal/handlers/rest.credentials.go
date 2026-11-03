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

// CredentialsUpdate is the JSON body answering a password or email change: the account, and the
// token pair that keeps the caller's session past the change, absent when no session of the account
// made the request.
type CredentialsUpdate struct {
	Credentials

	Token *Token `json:"token,omitempty"`
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

func loadCredentialsUpdate(s *core.CredentialsUpdate) CredentialsUpdate {
	res := CredentialsUpdate{Credentials: loadCredentials(&s.Credentials)}

	if s.Token != nil {
		token := loadToken(s.Token)
		res.Token = &token
	}

	return res
}
