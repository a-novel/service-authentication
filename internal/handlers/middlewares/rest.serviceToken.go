package middlewares

import (
	"net/http"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/servicetoken"
)

// RestServiceToken returns the middleware that refuses a request unless its service token matches
// tokenConfig, and logs each refusal through logger. A request to a path listed in open passes
// unchecked, and client fetches the issuer's keys. Without an issuer, the middleware lets every
// request through.
//
// Wrap the whole router with it. The router then reads the client IP header and counts its rate
// limits for verified callers alone.
func RestServiceToken(
	tokenConfig servicetoken.Config, client *http.Client, logger logging.Log, open ...string,
) func(http.Handler) http.Handler {
	if tokenConfig.Issuer == "" {
		return func(next http.Handler) http.Handler { return next }
	}

	return httpf.ServiceTokenMiddleware(servicetoken.NewVerifier(tokenConfig, client), logger, open...)
}
