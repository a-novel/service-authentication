package core

import "github.com/google/uuid"

// RefreshTokenClaims is the JWT payload of a refresh token: the token's own JTI and
// the user it authenticates. The token-refresh flow matches Jti against an access
// token's RefreshTokenID to bind the pair. See AccessTokenClaims for the binding.
type RefreshTokenClaims struct {
	Jti    string    `json:"jti,omitempty"`
	UserID uuid.UUID `json:"userID,omitempty"`
}

// RefreshTokenClaimsForm is the payload submitted to the signer to mint a refresh
// token. The JTI is assigned by the signer, so only the user ID is supplied here.
type RefreshTokenClaimsForm struct {
	UserID uuid.UUID `json:"userID,omitempty"`
}
