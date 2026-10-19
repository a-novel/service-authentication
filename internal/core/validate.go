package core

import (
	"errors"
	"slices"

	"github.com/go-playground/validator/v10"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// ErrInvalidRequest is returned by every Exec entry point in this package when
// the request fails struct-level validation. It is joined onto the underlying
// validator error so callers can branch on it with errors.Is and surface the
// detailed validation message at the same time.
var ErrInvalidRequest = errors.New("invalid request")

// ValidateLang is a go-playground/validator field-level validator that accepts
// any language code listed in authconfig.KnownLangs. It is registered under the
// "langs" tag at package init.
func ValidateLang(fl validator.FieldLevel) bool {
	return slices.Contains(authconfig.KnownLangs, fl.Field().String())
}

func init() {
	err := errors.Join(
		validate.RegisterValidation("langs", ValidateLang),
		validate.RegisterValidation("role", ValidateCredentialsRole),
		validate.RegisterValidation("usage", ValidateShortCodeUsage),
	)
	if err != nil {
		panic(err)
	}
}
