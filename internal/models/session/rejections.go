package session

import (
	"github.com/a-novel-kit/jwt/v2"
	"github.com/a-novel-kit/jwt/v2/jwp"
	"github.com/a-novel-kit/jwt/v2/jws"
)

// TokenRejections lists the errors with which the claims verifier rejects a session token itself,
// such as a bad signature or a malformed header. The holder resolves them by refreshing or signing
// in again. Any other verifier error, such as an unreachable key source, is a server fault.
var TokenRejections = []error{
	jws.ErrInvalidSignature,
	jwp.ErrInvalidClaims,
	jwt.ErrUnsupportedTokenFormat,
	jwt.ErrTokenTooLarge,
	jwt.ErrMismatchRecipientPlugin,
	jwt.ErrMissingCritHeader,
	jwt.ErrUnsupportedCritHeader,
	jwt.ErrReplicatedClaimMismatch,
}
