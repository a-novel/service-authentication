package middlewares

import (
	"errors"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"

	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// ErrClientToken refuses an access token that an OAuth client holds, on a route that manages an
// account.
var ErrClientToken = errors.New("access token held by a client")

// RestRefuseClientTokens returns the middleware that answers 403 to a request whose access token an
// OAuth client holds. A client signs a person in on its own platform, and an account is managed on
// the account platform alone, so a client's token reaches no route that manages accounts (RFC 9700
// §2.3). Mount it after [Auth], which puts the claims on the context. A request without claims
// passes.
func RestRefuseClientTokens(logger logging.Log) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			claims, err := session.GetClaimsContext(ctx)
			if err == nil && claims != nil && claims.ClientID != "" {
				err = fmt.Errorf("%w: %q", ErrClientToken, claims.ClientID)
			}

			if err != nil {
				errMap := httpf.ErrMap{ErrClientToken: http.StatusForbidden}
				httpf.HandleError(ctx, logger, w, trace.SpanFromContext(ctx), errMap, err)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
