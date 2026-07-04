package core

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/v2/internal/config"
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

// ValidateCredentialsRole is a go-playground/validator field-level validator that
// accepts a string only when it names a role defined in the default permissions
// configuration. It is registered under the "role" tag at package init.
func ValidateCredentialsRole(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for role := range config.PermissionsConfigDefault.Roles {
		if val == role {
			return true
		}
	}

	return false
}
