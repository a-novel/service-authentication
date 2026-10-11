package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
)

// RestLimits holds the strict limits that guard routes issuing tokens, checking passwords or
// sending email, on top of the standard one. Mount them after the auth middleware, so a signed-in
// caller counts as its user rather than its visitor IP.
type RestLimits struct {
	limits   httpf.Limits
	policies config.Limits
}

// NewRestLimits builds the configured policies on limits, which may disable them for test stacks.
func NewRestLimits(limits httpf.Limits, policies config.Limits) *RestLimits {
	return &RestLimits{limits: limits, policies: policies}
}

// Strict caps each caller of a route issuing anonymous tokens or invitation requests.
func (limits *RestLimits) Strict() func(http.Handler) http.Handler {
	return limits.by(limits.policies.Strict, restCallerKey)
}

// Login caps each visitor's logins over a short and a longer window, and the logins naming one
// email over the longer one.
func (limits *RestLimits) Login() func(http.Handler) http.Handler {
	return chi.Chain(
		limits.by(limits.policies.LoginBurst, restCallerKey),
		limits.by(limits.policies.Login, restCallerKey),
		limits.by(limits.policies.Login, restEmailKey),
	).Handler
}

// Mail applies the strict limit to each caller, and the mailbox limit to each address.
func (limits *RestLimits) Mail() func(http.Handler) http.Handler {
	return chi.Chain(limits.Strict(), limits.Mailbox()).Handler
}

// Mailbox caps the emails each address receives from one route, whoever asks for them, so many
// callers cannot gang up on one mailbox.
func (limits *RestLimits) Mailbox() func(http.Handler) http.Handler {
	return limits.by(limits.policies.Mailbox, restEmailKey)
}

func (limits *RestLimits) by(rate httpf.Rate, key httprate.KeyFunc) func(http.Handler) http.Handler {
	return limits.limits.By(rate.Requests, rate.Window, key)
}

// restCallerKey counts the signed-in user, else the visitor IP.
func restCallerKey(r *http.Request) (string, error) {
	claims, _ := r.Context().Value(ClaimsContextKey{}).(*core.AccessTokenClaims)
	if claims != nil && claims.UserID != nil {
		return "user:" + claims.UserID.String(), nil
	}

	return httpf.VisitorKey(r)
}

// restEmailKey reads the body, then puts it back for the handler. Letter case does not open a
// fresh budget for the same mailbox.
func restEmailKey(r *http.Request) (string, error) {
	body, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))

	var request struct {
		Email string `json:"email"`
	}

	if err == nil {
		// The handler refuses a malformed body; until then it shares the empty email's budget.
		_ = json.Unmarshal(body, &request)
	}

	return "email:" + strings.ToLower(strings.TrimSpace(request.Email)), nil
}
