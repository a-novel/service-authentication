package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// Each policy below allows a number of requests per window.
const (
	standardRequests = 5

	strictRequests = 3
	strictWindow   = 5 * time.Minute

	// A login asks for retries: a forgotten password takes a few attempts.
	loginBurstRequests = 10
	loginRequests      = 30
	loginWindow        = 10 * time.Minute
)

// RestLimits builds the rate limits mounted on routes. Each refuses excess requests with 429 and
// Retry-After; disabled limits let every request through, for test stacks.
//
// The strict limits guard routes that issue tokens or send email. Mount them after the auth
// middleware, so a signed-in caller counts as its user rather than its visitor IP.
type RestLimits struct {
	enabled bool
}

// NewRestLimits returns the limits, refusing excess requests only when enabled.
func NewRestLimits(enabled bool) *RestLimits {
	return &RestLimits{enabled: enabled}
}

// Standard caps each caller at 5 requests per second on each route.
func (limits *RestLimits) Standard() func(http.Handler) http.Handler {
	return limits.caller(standardRequests, time.Second)
}

// Strict caps each caller at 3 requests per 5 minutes, for anonymous sessions and invitation
// requests.
func (limits *RestLimits) Strict() func(http.Handler) http.Handler {
	return limits.caller(strictRequests, strictWindow)
}

// Login lets each visitor try 10 logins per minute and 30 per 10 minutes, and each email be tried
// 30 times per 10 minutes.
func (limits *RestLimits) Login() func(http.Handler) http.Handler {
	return chain(
		limits.caller(loginBurstRequests, time.Minute),
		limits.caller(loginRequests, loginWindow),
		limits.email(loginRequests, loginWindow),
	)
}

// Mail caps each caller at 3 requests per 5 minutes, and each address at 3 emails per hour.
func (limits *RestLimits) Mail() func(http.Handler) http.Handler {
	return chain(limits.Strict(), limits.Mailbox())
}

// Mailbox caps each address at 3 emails per hour from one route, whoever asks for them.
func (limits *RestLimits) Mailbox() func(http.Handler) http.Handler {
	return limits.email(strictRequests, time.Hour)
}

// caller counts the signed-in user, else the visitor IP.
func (limits *RestLimits) caller(limit int, window time.Duration) func(http.Handler) http.Handler {
	return limits.by(limit, window, httprate.JoinKeys(restCallerKey, restRouteKey))
}

// email counts the email named in the JSON body, so many callers cannot gang up on one mailbox or
// account.
func (limits *RestLimits) email(limit int, window time.Duration) func(http.Handler) http.Handler {
	return limits.by(limit, window, httprate.JoinKeys(restEmailKey, restRouteKey))
}

func (limits *RestLimits) by(limit int, window time.Duration, key httprate.KeyFunc) func(http.Handler) http.Handler {
	if !limits.enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	return httprate.LimitBy(limit, window, key)
}

func chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return chi.Chain(middlewares...).Handler
}

// restCallerKey counts an IPv6 visitor per /64, the block one client usually controls.
func restCallerKey(r *http.Request) (string, error) {
	claims, _ := r.Context().Value(ClaimsContextKey{}).(*core.AccessTokenClaims)
	if claims != nil && claims.UserID != nil {
		return "user:" + claims.UserID.String(), nil
	}

	return "visitor:" + httprate.CanonicalizeIP(middleware.GetClientIP(r.Context())), nil
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

// restRouteKey gives each route its own budget, shared by all its path parameters.
func restRouteKey(r *http.Request) (string, error) {
	return r.Method + " " + chi.RouteContext(r.Context()).RoutePattern(), nil
}
