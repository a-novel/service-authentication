package services

import (
	"errors"

	"github.com/go-playground/validator/v10"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

var ErrInvalidRequest = errors.New("invalid request")

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
