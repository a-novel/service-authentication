package services

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type ShortCode struct {
	ID uuid.UUID

	Usage  string
	Target string
	Data   []byte

	CreatedAt time.Time
	ExpiresAt time.Time

	PlainCode string
}

const (
	ShortCodeUsageValidateEmail = "validateEmail"
	ShortCodeUsageResetPassword = "resetPassword"
	ShortCodeUsageRegister      = "register"
)

var KnownShortCodeUsages = []string{
	ShortCodeUsageValidateEmail,
	ShortCodeUsageResetPassword,
	ShortCodeUsageRegister,
}

func ValidateShortCodeUsage(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, usage := range KnownShortCodeUsages {
		if val == usage {
			return true
		}
	}

	return false
}
