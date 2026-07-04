package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ShortCode is a validation mechanism. It stores randomly generated, temporary
// passwords used to authorize tasks that must not require direct authentication.
//
// Like passwords, short codes are stored with one-way encryption: the original value
// cannot be recovered from the stored one. The clear value is sent to the client
// through a secure channel.
//
// A short code may only be used once, and is discarded after a successful use.
//
// Unlike most other entities, the uniqueness constraint applies only to non-expired,
// non-deleted short codes, so it naturally evolves over time.
type ShortCode struct {
	bun.BaseModel `bun:"table:short_codes"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// The encrypted version of the generated short code, used to perform password-like
	// validation.
	Code string `bun:"code"`

	// Usage identifies the flow the short code is valid for. Together with Target it
	// forms the pair that is unique among active short codes.
	Usage string `bun:"usage"`
	// Target identifies the subject of the operation, usually the channel the clear code
	// was sent to. See Usage for the uniqueness rule on the pair.
	Target string `bun:"target"`

	// Additional data to use after validating a short code.
	Data []byte `bun:"data"`

	CreatedAt time.Time `bun:"created_at"`
	// ExpiresAt bounds the short code's lifetime. Unlike a password, the clear value
	// lives wherever it was delivered, so the code is kept as short-lived as possible —
	// usually a matter of days — to limit exposure, since it can authorize critical
	// operations such as a password reset.
	ExpiresAt time.Time `bun:"expires_at"`

	// DeletedAt marks a short code invalidated before its expiration, either manually
	// (an admin suppressing a leak) or naturally when the code is consumed or superseded
	// by a newer code for the same Target / Usage pair. DeletedComment records the reason.
	DeletedAt      *time.Time `bun:"deleted_at"`
	DeletedComment *string    `bun:"deleted_comment"`
}
