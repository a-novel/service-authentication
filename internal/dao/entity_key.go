package dao

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/a-novel/authentication/models"
)

var ErrKeyNotFound = errors.New("key not found")

// KeyEntity represents a public/private key pair used for Signature and Encryption purposes.
//
// A given key pair is REQUIRED to have an expiration date, as it must be rotated on a regular basis. Only public keys
// may be exposed to the application.
type KeyEntity struct {
	bun.BaseModel `bun:"table:keys,select:active_keys"`

	// Unique identifier of the key.
	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// The private key in JSON Web Key format.
	//
	// The key MUST BE encrypted, and the result of this encryption is stored as a base64 raw URL encoded string.
	PrivateKey string `bun:"private_key"`
	// The public key in JSON Web Key format. The key is stored as a base64 raw URL encoded string.
	//
	// This value is OPTIONAL for symmetric keys.
	PublicKey *string `bun:"public_key"`

	// Intended usage of the key. See the type documentation for more details.
	Usage models.KeyUsage `bun:"usage"`

	// Time at which the key was created. This is important when listing keys, as the most recent keys are
	// used in priority.
	CreatedAt time.Time `bun:"created_at"`
	// Expiration of the key. Each key pair is REQUIRED to expire past a certain time. Once the expiration date
	// is reached, the key pair becomes invisible to the keys view.
	ExpiresAt time.Time `bun:"expires_at"`

	// Time at which the key is marked as deleted. This field is the result of a manual, unexpected deletion
	// consecutive to an abnormal event. Once this field is set, the key is no longer visible to the keys view.
	DeletedAt *time.Time `bun:"deleted_at"`
	// Comment explaining why the key was deleted. This field is set when the key is marked as deleted.
	DeletedComment *string `bun:"deleted_comment"`
}
