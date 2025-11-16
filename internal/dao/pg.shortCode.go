package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ShortCode is used as a validation mechanism. It stores randomly generated temporary
// passwords that can be used for various tasks, without requiring or without the possibility
// for direct authentication.
//
// Like passwords, short codes MUST be encrypted using one-way encryption (the original message
// cannot be reverse-engineered). The clear value of the short code should be sent to the
// client through a secure channel.
//
// A short-code should only ever be used once, It MUST BE discarded after a successful use.
//
// Unlike most other entities, unique constraints only apply to non-expired, non-deleted
// short codes. This mean those rules naturally evolve over time.
type ShortCode struct {
	bun.BaseModel `bun:"table:short_codes"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// The encrypted version of the generated short code, used to perform password-like
	// validation.
	Code string `bun:"code"`

	// Usage of the short code, used to identify a short code along with the Target field.
	// There can be only a single combination of Usage / Target among active short codes.
	Usage string `bun:"usage"`
	// Target of the short code, usually the channel used to send the clear code. It is
	// used to identify a short code along with the Usage field.
	// There can be only a single combination of Usage / Target among active short codes.
	Target string `bun:"target"`

	// Additional data to use after validating a short code.
	Data []byte `bun:"data"`

	CreatedAt time.Time `bun:"created_at"`
	// A short code, unlike a password, is sent in the wild (i.e. the clear version is
	// persisted at the place it was sent). It is then important to make this short code
	// as short-lived as possible, especially as it can be used to perform critical
	// operations (e.g. password reset).
	//
	// The ExpiresAt value should be as close as possible in the future, usually a matter of days.
	ExpiresAt time.Time `bun:"expires_at"`

	// DeletedAt means the short code has been invalidated BEFORE its expiration.
	// This can happen during a leakage (manual suppression by admins), but also happens
	// naturally when:
	//
	//  - A short code is consumed (it can only be used once).
	//  - A new short code with the same Target / Usage is created.
	//
	// The reason for a short code deletion is stated in the DeletedComment field.
	DeletedAt      *time.Time `bun:"deleted_at"`
	DeletedComment *string    `bun:"deleted_comment"`
}
