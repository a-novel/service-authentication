package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Session is one login on the account platform, joined by the OAuth clients that sign in through it.
// A session is active while [Session.ExpiresAt] is ahead and no [RefreshTokenRevocation] holds its
// ID. Otherwise it is logged out, and its row stays until the account removes it or the purge does.
type Session struct {
	bun.BaseModel `bun:"table:sessions"`

	// ID is the JTI of the refresh token that opened the session.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// UserID is the account the session signs in.
	UserID uuid.UUID `bun:"user_id,type:uuid"`

	// UserAgent is the User-Agent of the device's browser, empty when unknown.
	UserAgent string `bun:"user_agent"`
	// IPAddress is the IP address of the device, empty when not recorded.
	IPAddress string `bun:"ip_address,nullzero"`
	// Region is the CLDR region code of the device, such as FR, empty when not recorded.
	Region string `bun:"region,nullzero"`
	// City is the city of the device, empty when not recorded.
	City string `bun:"city,nullzero"`

	CreatedAt time.Time `bun:"created_at"`
	// LastActiveAt is when the session's latest access token was issued.
	LastActiveAt time.Time `bun:"last_active_at"`
	// ExpiresAt is when the session ends. It never changes.
	ExpiresAt time.Time `bun:"expires_at"`

	// LoggedOut is whether the session ended or was signed out. The queries that read sessions
	// compute it, and no write stores it.
	LoggedOut bool `bun:"logged_out,scanonly"`
}
