package services

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/config"
)

type Credentials struct {
	ID        uuid.UUID
	Email     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func ValidateCredentialsRole(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for role := range config.PermissionsConfigDefault.Roles {
		if val == role {
			return true
		}
	}

	return false
}
