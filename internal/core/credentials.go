package core

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// Credentials is a user account as the core layer exposes it: identity, email,
// current role, and audit timestamps. The stored password hash stays in the DAO
// layer and is never carried on this type.
type Credentials struct {
	ID        uuid.UUID
	Email     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CredentialsSelf is the caller's own account, as [CredentialsGetSelf] returns it.
type CredentialsSelf struct {
	Credentials

	// PendingEmail is the email change awaiting confirmation, nil when none does.
	PendingEmail *PendingEmail
}

// PendingEmail is an email change requested through [ShortCodeCreateEmailUpdate]. It lasts until
// the user confirms it, requests another change, or lets the confirmation code expire.
type PendingEmail struct {
	// Email is the new address, which takes effect once confirmed.
	Email string

	// ExpiresAt is when the confirmation code expires.
	ExpiresAt time.Time
}

// ValidateCredentialsRole is a go-playground/validator field-level validator that
// accepts a string only when it names a role defined in the default permissions
// configuration. It is registered under the "role" tag at package init.
func ValidateCredentialsRole(fl validator.FieldLevel) bool {
	_, ok := authconfig.PermissionsConfigDefault.Roles[fl.Field().String()]

	return ok
}

func newCredentials(entity *dao.Credentials) *Credentials {
	return &Credentials{
		ID:        entity.ID,
		Email:     entity.Email,
		Role:      entity.Role,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
	}
}
