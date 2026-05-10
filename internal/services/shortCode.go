package services

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// ShortCode is a one-time verification code issued for a sensitive identity
// operation (registration, password reset, email change). Callers receive it
// from [ShortCodeCreate]; a separate [ShortCodeConsume] call validates and
// retires it.
type ShortCode struct {
	ID uuid.UUID

	// Usage selects which flow this code is valid for; values come from the
	// ShortCodeUsage* constants. A code issued for one usage cannot be redeemed
	// for another.
	Usage string
	// Target identifies the subject of the operation — for email-bound flows,
	// this is the email address the code was sent to.
	Target string
	// Data carries flow-specific context the consumer needs (for example, the
	// new email for an email-change confirmation). Opaque to ShortCode itself.
	Data []byte

	CreatedAt time.Time
	ExpiresAt time.Time

	// PlainCode is the user-facing code emailed to the target. It is populated
	// only on the response from [ShortCodeCreate]; the database stores a hash.
	PlainCode string
}

const (
	// ShortCodeUsageValidateEmail gates an email-change confirmation: the code is
	// emailed to the prospective new address and consumed by [CredentialsUpdateEmail].
	ShortCodeUsageValidateEmail = "validateEmail"
	// ShortCodeUsageResetPassword gates a password reset: the code is emailed to
	// the credentials' current address and consumed by [CredentialsUpdatePassword].
	ShortCodeUsageResetPassword = "resetPassword"
	// ShortCodeUsageRegister gates a new-account registration: the code is emailed
	// to the prospective address and consumed by [CredentialsCreate].
	ShortCodeUsageRegister = "register"
)

// KnownShortCodeUsages enumerates every valid value for [ShortCode.Usage]. It
// backs the "short_code_usage" validator registered on the package validator
// instance.
var KnownShortCodeUsages = []string{
	ShortCodeUsageValidateEmail,
	ShortCodeUsageResetPassword,
	ShortCodeUsageRegister,
}

// ValidateShortCodeUsage is a go-playground/validator field-level validator that
// accepts any value listed in [KnownShortCodeUsages]. Register it under a tag
// (e.g. "short_code_usage") to validate request fields that carry a usage value.
func ValidateShortCodeUsage(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	for _, usage := range KnownShortCodeUsages {
		if val == usage {
			return true
		}
	}

	return false
}
