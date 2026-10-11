package middlewares

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/a-novel-kit/golib/httpf"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestLimits holds the strict limits that guard routes issuing tokens, checking passwords or
// sending email, on top of the standard one. Mount them after the auth middleware, so a signed-in
// caller counts as its user rather than its visitor IP.
type RestLimits struct {
	limits   httpf.Limits
	policies authconfig.Limits
}

// NewRestLimits builds the configured policies on limits, which may disable them for test stacks.
func NewRestLimits(limits httpf.Limits, policies authconfig.Limits) *RestLimits {
	return &RestLimits{limits: limits, policies: policies}
}

// Strict caps each caller of a route issuing anonymous tokens or invitation requests.
func (limits *RestLimits) Strict() func(http.Handler) http.Handler {
	return limits.limits.By(limits.policies.Strict, CallerKey)
}

// Login caps each visitor's logins over a short and a longer window, and the logins naming one
// email over the longer one.
func (limits *RestLimits) Login() func(http.Handler) http.Handler {
	return chi.Chain(
		limits.limits.By(limits.policies.LoginBurst, CallerKey),
		limits.limits.By(limits.policies.Login, CallerKey),
		limits.limits.By(limits.policies.Login, emailKey),
	).Handler
}

// Mail applies the strict limit to each caller, and the mailbox limit to each address.
func (limits *RestLimits) Mail() func(http.Handler) http.Handler {
	return chi.Chain(limits.Strict(), limits.Mailbox()).Handler
}

// Mailbox caps the emails each address receives from one route, whoever asks for them, so many
// callers cannot gang up on one mailbox.
func (limits *RestLimits) Mailbox() func(http.Handler) http.Handler {
	return limits.limits.By(limits.policies.Mailbox, emailKey)
}

// emailKey counts the email a request names in its body.
var emailKey = httpf.JSONFieldKey("email")

// CallerKey keys a rate limit by the signed-in user, else by the visitor IP. Mount the limit after
// the auth middleware, which signs the user in.
func CallerKey(r *http.Request) (string, error) {
	claims, _ := r.Context().Value(ClaimsContextKey{}).(*core.AccessTokenClaims)
	if claims != nil && claims.UserID != nil {
		return "user:" + claims.UserID.String(), nil
	}

	return httpf.VisitorKey(r)
}
