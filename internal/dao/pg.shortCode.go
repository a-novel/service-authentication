package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ShortCode is a randomly generated, temporary password authorizing a task that must
// run without direct authentication.
//
// Short codes are stored hashed, so the original value cannot be recovered from the
// row; the clear value is delivered to the client over a secure channel. A code may
// be used once and is discarded on a successful use.
//
// The uniqueness constraint covers only active codes — unexpired and undeleted — so
// the set it applies to evolves over time.
type ShortCode struct {
	bun.BaseModel `bun:"table:short_codes"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// Code is the Argon2id hash of the generated short code, verified like a password.
	Code string `bun:"code"`

	// Usage identifies the flow the short code is valid for. Together with Target it
	// forms the pair that is unique among active short codes.
	Usage string `bun:"usage"`
	// Target identifies the subject of the operation, usually the channel the clear code
	// was sent to. See Usage for the uniqueness rule on the pair.
	Target string `bun:"target"`

	// Data carries flow-specific context used after validating the short code.
	Data []byte `bun:"data"`

	CreatedAt time.Time `bun:"created_at"`
	// ExpiresAt bounds the short code's lifetime. The clear value sits wherever it was
	// delivered and can authorize a critical operation such as a password reset, so
	// lifetimes are kept short — usually a matter of days.
	ExpiresAt time.Time `bun:"expires_at"`

	// DeletedAt marks a short code invalidated before its expiration: consumed,
	// superseded by a newer code for the same Target / Usage pair, or manually
	// suppressed by an admin after a leak. DeletedComment records the reason.
	DeletedAt      *time.Time `bun:"deleted_at"`
	DeletedComment *string    `bun:"deleted_comment"`
}
