package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// MailOutboxEntry is the mail of one short code, stored until a send delivers it. It holds what the
// mail needs except the plaintext code, which is never stored.
type MailOutboxEntry struct {
	bun.BaseModel `bun:"table:mail_outbox"`

	// ShortCodeID identifies the short code the mail carries.
	ShortCodeID uuid.UUID `bun:"short_code_id,pk,type:uuid"`

	// Usage is the short code's usage, which selects the mail.
	Usage string `bun:"usage"`
	// Target is the short code's target, which the mail's link carries.
	Target string `bun:"target"`

	// Recipient is the address the mail goes to.
	Recipient string `bun:"recipient"`
	// Lang is the language of the mail.
	Lang string `bun:"lang"`

	// Attempts counts the attempts claimed so far.
	Attempts int `bun:"attempts"`
	// DueAt is when the next attempt may start. A claim moves it past the attempt, so no other attempt
	// starts while one runs.
	DueAt time.Time `bun:"due_at"`
	// ExpiresAt is the expiry of the entry's first code. Retries stop there.
	ExpiresAt time.Time `bun:"expires_at"`
}
