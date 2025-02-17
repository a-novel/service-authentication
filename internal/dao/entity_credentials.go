package dao

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

var (
	ErrCredentialsAlreadyExists = errors.New("credentials already exists")
	ErrCredentialsNotFound      = errors.New("credentials not found")
)

// CredentialsEntity represents a set of credentials a user can use to authenticate with a non-anonymous session.
//
// A non-anonymous session grants more rights, and allows tracking of the user's activity.
type CredentialsEntity struct {
	bun.BaseModel `bun:"table:credentials"`

	// Unique identifier of the credentials.
	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// The user's email. This value MUST be unique, and is also used as a way to reach the user. An account cannot be
	// created / used unless the email has been verified first.
	Email string `bun:"email"`
	// The user's password, encrypted using scrypt. This will be used when creating a session.
	//
	// This value is sensitive, and MUST always be transmitted using a secure channel / encryption. It MUST NEVER be
	// stored clear in the database.
	//
	// This value may be empty when creating a shadow user. Shadow Users are users whose account is created indirectly,
	// and so their password must be set post-creation.
	// While technically valid, shadow users accounts cannot be used to create sessions, as this process requires a
	// password.
	Password string `bun:"password"`

	// Time at which the credentials were created.
	CreatedAt time.Time `bun:"created_at"`
	// Time at which the credentials were last updated.
	UpdatedAt time.Time `bun:"updated_at"`
}
