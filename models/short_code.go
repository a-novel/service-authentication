package models

import (
	"time"

	"github.com/google/uuid"
)

// ShortCode is a temporary password used to grant a user one-time access to a specific resource. Once a resource
// is accessed using a short code, the short code is expired.
type ShortCode struct {
	ID uuid.UUID

	// Information about the resource the short code grants access to.
	Usage ShortCodeUsage
	// The target that is allowed to access the resource. Only this target can retrieve the short code.
	Target string
	// Data used for the targeted resource. It can contain any information required to perform a specific action.
	Data []byte

	// Time at which the short code was created.
	CreatedAt time.Time
	// Expiration of the short code. Each short code is REQUIRED to expire past a certain time. Once the expiration date
	// is reached, the short code can no longer be used or retrieved.
	ExpiresAt time.Time

	// PlainCode is the clear password sent to the target via a secure channel.
	PlainCode string
}
