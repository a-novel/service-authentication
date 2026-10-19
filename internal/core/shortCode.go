package core

import (
	"slices"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// ShortCode is a one-time verification code issued for a sensitive identity
// operation (registration, password reset, email change). Callers receive it
// from [ShortCodeCreate]; [ShortCodeVerify] checks a submitted code and
// [ShortCodeConsume] retires it.
type ShortCode struct {
	ID uuid.UUID

	// Usage selects which flow this code is valid for; values come from the
	// ShortCodeUsage* constants. A code issued for one usage cannot be redeemed
	// for another.
	Usage string
	// Target identifies the subject of the operation: the email address of a
	// registration, the user ID otherwise.
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
// backs the "usage" validator tag registered on the package validator instance
// (see [ValidateShortCodeUsage]).
var KnownShortCodeUsages = []string{
	ShortCodeUsageValidateEmail,
	ShortCodeUsageResetPassword,
	ShortCodeUsageRegister,
}

// ValidateShortCodeUsage is the go-playground/validator field-level validator
// that accepts any value listed in [KnownShortCodeUsages]. The package's init()
// registers it under the "usage" tag, so request structs gate this with
// `validate:"required,usage"`.
func ValidateShortCodeUsage(fl validator.FieldLevel) bool {
	return slices.Contains(KnownShortCodeUsages, fl.Field().String())
}

func newShortCode(entity *dao.ShortCode, plainCode string) *ShortCode {
	return &ShortCode{
		ID:        entity.ID,
		Usage:     entity.Usage,
		Target:    entity.Target,
		Data:      entity.Data,
		CreatedAt: entity.CreatedAt,
		ExpiresAt: entity.ExpiresAt,
		PlainCode: plainCode,
	}
}
