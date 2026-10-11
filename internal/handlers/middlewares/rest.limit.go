package middlewares

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/a-novel-kit/golib/httpf"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

// Each policy below allows a number of requests per window.
const (
	strictRequests = 3
	strictWindow   = 5 * time.Minute

	// A login asks for retries: a forgotten password takes a few attempts.
	loginBurstRequests = 10
	loginRequests      = 30
	loginWindow        = 10 * time.Minute
)

// RestLimits holds the strict limits that guard routes issuing tokens, checking passwords or
// sending email, on top of the standard one. Mount them after the auth middleware, so a signed-in
// caller counts as its user rather than its visitor IP.
type RestLimits struct {
	limits httpf.Limits
}

// NewRestLimits builds the strict limits on limits, which may disable them for test stacks.
func NewRestLimits(limits httpf.Limits) *RestLimits {
	return &RestLimits{limits: limits}
}

// Strict caps each caller at 3 requests per 5 minutes, for anonymous sessions and invitation
// requests.
func (limits *RestLimits) Strict() func(http.Handler) http.Handler {
	return limits.limits.By(strictRequests, strictWindow, restCallerKey)
}

// Login lets each visitor try 10 logins per minute and 30 per 10 minutes, and each email be tried
// 30 times per 10 minutes.
func (limits *RestLimits) Login() func(http.Handler) http.Handler {
	return chi.Chain(
		limits.limits.By(loginBurstRequests, time.Minute, restCallerKey),
		limits.limits.By(loginRequests, loginWindow, restCallerKey),
		limits.limits.By(loginRequests, loginWindow, restEmailKey),
	).Handler
}

// Mail caps each caller at 3 requests per 5 minutes, and each address at 3 emails per hour.
func (limits *RestLimits) Mail() func(http.Handler) http.Handler {
	return chi.Chain(limits.Strict(), limits.Mailbox()).Handler
}

// Mailbox caps each address at 3 emails per hour from one route, whoever asks for them, so many
// callers cannot gang up on one mailbox.
func (limits *RestLimits) Mailbox() func(http.Handler) http.Handler {
	return limits.limits.By(strictRequests, time.Hour, restEmailKey)
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
