package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Credentials hold the information used to authenticate and identify a user.
type Credentials struct {
	bun.BaseModel `bun:"table:credentials"`

	// ID of the user, used to look up its information across the platform.
	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// Email the user authenticates with, unique across all users.
	Email string `bun:"email"`
	// Password is the Argon2id hash of the user's password; the plaintext is never
	// stored.
	Password string `bun:"password"`

	// Role determines which actions the user is allowed to take.
	Role string `bun:"role"`

	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}
