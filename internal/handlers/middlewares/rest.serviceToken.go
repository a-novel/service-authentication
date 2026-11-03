package middlewares

import (
	"errors"
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/servicetoken"
)

// ErrServiceTokenIssuerRequired refuses a server configured with no service-token issuer: it would
// take every request.
var ErrServiceTokenIssuerRequired = errors.New("service token issuer required")

// RestServiceToken returns the middleware that refuses a request unless its service token matches
// tokenConfig, and logs each refusal through logger. A request to a path listed in open passes
// unchecked, and client fetches the issuer's keys.
//
// The gate fails closed: without an issuer it returns [ErrServiceTokenIssuerRequired], so a
// deployment that forgets the issuer stops at startup instead of admitting every caller. Only
// issuerOptional, which a local or test stack sets, turns that into a middleware that lets every
// request through.
//
// Wrap the whole router with it. The router then reads the client IP header and counts its rate
// limits for verified callers alone.
func RestServiceToken(
	tokenConfig servicetoken.Config, issuerOptional bool, client *http.Client, logger logging.Log, open ...string,
) (func(http.Handler) http.Handler, error) {
	if tokenConfig.Issuer == "" {
		if issuerOptional {
			return func(next http.Handler) http.Handler { return next }, nil
		}

		return nil, ErrServiceTokenIssuerRequired
	}

	return httpf.ServiceTokenMiddleware(servicetoken.NewVerifier(tokenConfig, client), logger, open...), nil
}
