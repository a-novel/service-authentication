package middlewares

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/a-novel-kit/golib/httpf"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// RestLimits are the strict limits on routes that issue tokens, check passwords or send email, on
// top of the router's standard one. Mount each after the auth middleware, with chi's With, so a
// signed-in caller counts as its user rather than its visitor IP.
type RestLimits struct {
	// Anonymous caps each caller's anonymous sessions.
	Anonymous func(http.Handler) http.Handler
	// AuthorizationCode caps each signed-in user's authorization codes as Password caps password
	// changes.
	AuthorizationCode func(http.Handler) http.Handler
	// OAuthToken caps the token requests each client sends for each visitor.
	OAuthToken func(http.Handler) http.Handler
	// Login caps each visitor's logins over a short and a longer window, and the logins naming
	// one email over the longer one.
	Login func(http.Handler) http.Handler
	// Mail caps each caller's requests for an email, and the emails each address receives.
	Mail func(http.Handler) http.Handler
	// Mailbox caps the emails each address receives, whoever asks for them.
	Mailbox func(http.Handler) http.Handler
	// Password caps each signed-in user's password changes as Login caps logins.
	Password func(http.Handler) http.Handler
	// ShortCode caps each caller's password resets and email updates as Login caps logins, and
	// those naming one account over the longer window.
	ShortCode func(http.Handler) http.Handler
	// SignUp caps each caller's sign-ups as Login caps logins, and those naming one email over the
	// longer window.
	SignUp func(http.Handler) http.Handler
	// Waitlist caps the waitlist requests waiting on Google, their size, and each caller's requests.
	Waitlist func(http.Handler) http.Handler
}

// NewRestLimits builds the configured policies, which the REST config may disable for test stacks.
func NewRestLimits(rest httpf.ServerConfig, policies authconfig.Limits) RestLimits {
	limits := httpf.NewLimits(rest)

	strict := limits.By(policies.Strict, CallerKey)
	mailbox := limits.By(policies.Mailbox, emailKey)
	loginBurst := limits.By(policies.LoginBurst, CallerKey)
	login := limits.By(policies.Login, CallerKey)
	loginEmail := limits.By(policies.Login, emailKey)
	loginAccount := limits.By(policies.Login, accountKey)
	waitlistConcurrency := httpf.LimitConcurrency(policies.WaitlistMaxConcurrent)
	waitlistSize := middleware.RequestSize(policies.WaitlistMaxRequestSize)
	oauthToken := limits.By(policies.OAuthToken, ClientVisitorKey)

	return RestLimits{
		Anonymous:         strict,
		AuthorizationCode: chi.Chain(loginBurst, login).Handler,
		OAuthToken:        oauthToken,
		Login:             chi.Chain(loginBurst, login, loginEmail).Handler,
		Mail:              chi.Chain(strict, mailbox).Handler,
		Mailbox:           mailbox,
		Password:          chi.Chain(loginBurst, login).Handler,
		ShortCode:         chi.Chain(loginBurst, login, loginAccount).Handler,
		SignUp:            chi.Chain(loginBurst, login, loginEmail).Handler,
		Waitlist:          chi.Chain(waitlistConcurrency, waitlistSize, strict).Handler,
	}
}

var (
	// emailKey counts the email a request names in its body.
	emailKey = httpf.BodyKeyRate("email")
	// accountKey counts the account a request names in its body by ID.
	accountKey = httpf.BodyKeyRate("userID")
)

// ClientVisitorKey keys a rate limit by the calling service, as its verified service token names it,
// and the visitor it calls for. Each visitor of a client then spends a budget of its own, and one
// visitor cannot exhaust the client's.
func ClientVisitorKey(r *http.Request) (string, error) {
	visitor, err := httpf.VisitorKey(r)
	if err != nil {
		return "", err
	}

	caller, _ := httpf.ServiceTokenCaller(r.Context())

	return "service:" + caller + ":" + visitor, nil
}

// CallerKey keys a rate limit by the signed-in user, else by the visitor IP. Mount the limit after
// the auth middleware, which signs the user in.
func CallerKey(r *http.Request) (string, error) {
	claims, _ := r.Context().Value(session.ClaimsContextKey{}).(*session.Claims)
	if claims != nil && claims.UserID != nil {
		return "user:" + claims.UserID.String(), nil
	}

	return httpf.VisitorKey(r)
}
