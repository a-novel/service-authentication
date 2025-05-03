package dao

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/a-novel/service-authentication/models"
)

var (
	ErrShortCodeAlreadyExists = errors.New("short code already exists")
	ErrShortCodeNotFound      = errors.New("short code not found")
)

// ShortCodeEntity represents a temporary password used to grant one-time access to a specific resource.
// Once a resource is accessed using a short code, the short code MUST be expired.
//
// Only ONE version of a short code is allowed to exist for a given target and usage at a given time. This does not
// include expired or deleted short codes.
//
// Under normal circumstances, the combination of target and usage should be used to retrieve a short code, as this
// combination is guaranteed to point to a single, active short code.
type ShortCodeEntity struct {
	bun.BaseModel `bun:"table:short_codes,select:active_short_codes"`

	// Unique identifier of the short code.
	ID uuid.UUID `bun:"id,pk,type:uuid"`

	// The encrypted code. A clear version of this code is sent to the target.
	Code string `bun:"code"`

	// Information about the resource the short code grants access to.
	Usage models.ShortCodeUsage `bun:"usage"`
	// The target that is allowed to access the resource. Only this target can retrieve the short code.
	Target string `bun:"target"`

	// Data used for the targeted resource. It can contain any information required to perform a specific action.
	Data []byte `bun:"data"`

	// Time at which the short code was created.
	CreatedAt time.Time `bun:"created_at"`
	// Expiration of the short code. Each short code is REQUIRED to expire past a certain time. Once the expiration date
	// is reached, the short code can no longer be used or retrieved.
	ExpiresAt time.Time `bun:"expires_at"`

	// Time at which the short code was deleted. A short code MUST be deleted under normal circumstances, when the
	// resource it grants access to has been consumed.
	//
	// Other events may lead to the deletion of a short code. Those events are described in the DeletedComment field.
	DeletedAt *time.Time `bun:"deleted_at"`
	// The comment gives more insights about the reason a short code was deleted. This comment is required, and under
	// normal circumstances, MUST be a static, generic value to facilitate the understanding of the deletion.
	//
	// Available values for this field are:
	//  - DeleteCommentOverrideWithNewerKey
	//  - DeleteCommentConsumed
	DeletedComment *string `bun:"deleted_comment"`
}
