package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Credentials represent the information used to authenticate and discriminate a user.
type Credentials struct {
	bun.BaseModel `bun:"table:credentials"`

	// ID of the user. It can be used to retrieve public or private information about it.
	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// Email used by the user for authentication. Must be unique across all users.
	Email string `bun:"email"`
	// Password used to verify the user identity. It must be encrypted securely.
	Password string `bun:"password"`

	// The role of a user determines which actions it will be allowed to take.
	Role string `bun:"role"`

	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}
