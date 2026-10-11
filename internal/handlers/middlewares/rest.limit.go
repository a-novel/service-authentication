package middlewares

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/a-novel-kit/golib/httpf"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestLimits are the strict limits on routes that issue tokens, check passwords or send email, on
// top of the router's standard one. Mount each after the auth middleware, with chi's With, so a
// signed-in caller counts as its user rather than its visitor IP.
type RestLimits struct {
	// Anonymous caps each caller's anonymous sessions.
	Anonymous func(http.Handler) http.Handler
	// Login caps each visitor's logins over a short and a longer window, and the logins naming
	// one email over the longer one.
	Login func(http.Handler) http.Handler
	// Mail caps each caller's requests for an email, and the emails each address receives.
	Mail func(http.Handler) http.Handler
	// Mailbox caps the emails each address receives, whoever asks for them.
	Mailbox func(http.Handler) http.Handler
	// Waitlist caps the waitlist requests waiting on Google, their size, and each caller's requests.
	Waitlist func(http.Handler) http.Handler
}

// NewRestLimits builds the configured policies, which the REST config may disable for test stacks.
func NewRestLimits(rest httpf.ServerConfig, policies authconfig.Limits) RestLimits {
	limits := httpf.NewLimits(rest)
	strict := limits.By(policies.Strict, CallerKey)
	mailbox := limits.By(policies.Mailbox, emailKey)

	return RestLimits{
		Anonymous: strict,
		Login: chi.Chain(
			limits.By(policies.LoginBurst, CallerKey),
			limits.By(policies.Login, CallerKey),
			limits.By(policies.Login, emailKey),
		).Handler,
		Mail:    chi.Chain(strict, mailbox).Handler,
		Mailbox: mailbox,
		Waitlist: chi.Chain(
			httpf.LimitConcurrency(policies.WaitlistMaxConcurrent),
			middleware.RequestSize(policies.WaitlistMaxRequestSize),
			strict,
		).Handler,
	}
}

// emailKey counts the email a request names in its body.
var emailKey = httpf.BodyKeyRate("email")

// CallerKey keys a rate limit by the signed-in user, else by the visitor IP. Mount the limit after
// the auth middleware, which signs the user in.
func CallerKey(r *http.Request) (string, error) {
	claims, _ := r.Context().Value(ClaimsContextKey{}).(*core.AccessTokenClaims)
	if claims != nil && claims.UserID != nil {
		return "user:" + claims.UserID.String(), nil
	}

	return httpf.VisitorKey(r)
}
