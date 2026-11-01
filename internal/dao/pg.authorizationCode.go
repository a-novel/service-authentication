package dao

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// AuthorizationCode is a one-time code the account platform obtained for one client, to sign a
// person in there within the account platform's session. The client trades it for a token pair in
// that session, with the PKCE verifier only it holds (RFC 7636).
//
// The code itself is never stored: its SHA-256 identifies the row. An exchange marks the row
// consumed rather than deleting it, so a second exchange of the same code finds it.
type AuthorizationCode struct {
	bun.BaseModel `bun:"table:authorization_codes"`

	// ID is the SHA-256 of the code.
	ID []byte `bun:"id,pk"`
	// UserID is the account the code signs in.
	UserID uuid.UUID `bun:"user_id,type:uuid"`

	// ClientID is the client the code was issued to, as the client registry names it.
	ClientID string `bun:"client_id"`
	// RedirectURI is the URI the code was sent to: the one the authorization request named, or the
	// client's.
	RedirectURI string `bun:"redirect_uri"`
	// RedirectURIInRequest reports whether the authorization request named the redirect URI. The token
	// request must then name it too.
	RedirectURIInRequest bool `bun:"redirect_uri_in_request"`
	// CodeChallenge is the PKCE challenge of the authorization request.
	CodeChallenge string `bun:"code_challenge"`
	// CodeChallengeMethod is the PKCE method that derives the challenge from the client's code
	// verifier, S256 alone.
	CodeChallengeMethod string `bun:"code_challenge_method"`

	// SessionID is the session the exchanged token pair joins.
	SessionID string `bun:"session_id"`
	// SessionExpiresAt is when that session ends.
	SessionExpiresAt time.Time `bun:"session_expires_at"`

	// ExpiresAt is the last instant the code exchanges.
	ExpiresAt time.Time `bun:"expires_at"`
	// ConsumedAt is when the code was exchanged, nil until then.
	ConsumedAt *time.Time `bun:"consumed_at"`
}
