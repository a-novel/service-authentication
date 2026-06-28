package core

import (
	"errors"

	"github.com/go-playground/validator/v10"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// ErrInvalidRequest is returned by every Exec entry point in this package when
// the request fails struct-level validation. It is joined onto the underlying
// validator error so callers can branch on it with errors.Is and surface the
// detailed validation message at the same time.
var ErrInvalidRequest = errors.New("invalid request")

// ValidateLang is a go-playground/validator field-level validator that accepts
// any language code listed in config.KnownLangs. It is registered under the
// "langs" tag at package init.
func ValidateLang(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, lang := range config.KnownLangs {
		if val == lang {
			return true
		}
	}

	return false
}

func init() {
	err := validate.RegisterValidation("langs", ValidateLang)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("role", ValidateCredentialsRole)
	if err != nil {
		panic(err)
	}

	err = validate.RegisterValidation("usage", ValidateShortCodeUsage)
	if err != nil {
		panic(err)
	}
}
