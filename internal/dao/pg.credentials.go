package dao

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	// ErrCredentialsNotFound is returned when no credentials match the request.
	ErrCredentialsNotFound = errors.New("credentials not found")
	// ErrCredentialsAlreadyExists is returned when another account already holds the email.
	ErrCredentialsAlreadyExists = errors.New("credentials already exists")
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
	// LocationConsent is whether the account records the location of its sessions.
	LocationConsent bool `bun:"location_consent"`

	// CreatedAt is when the account was registered.
	CreatedAt time.Time `bun:"created_at"`
	// UpdatedAt is when the email, password or role last changed.
	UpdatedAt time.Time `bun:"updated_at"`
}
