package dao

import (
	"time"

	"github.com/uptrace/bun"
)

// RefreshTokenRevocation records a revoked refresh token. A refresh token is a stateless JWT, so its
// revocation is the only server-side trace of a session that ended before the token expired.
type RefreshTokenRevocation struct {
	bun.BaseModel `bun:"table:refresh_token_revocations"`

	// JTI is the revoked refresh token's unique identifier.
	JTI string `bun:"jti,pk"`

	// ExpiresAt is the last instant the verifier accepts the refresh token. The row has no use past
	// it, and [PgRefreshTokenRevocationPurge] deletes it.
	ExpiresAt time.Time `bun:"expires_at"`
	// CreatedAt is when the refresh token was first revoked.
	CreatedAt time.Time `bun:"created_at"`
}
