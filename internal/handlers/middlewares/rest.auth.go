package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/samber/lo"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/logging"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// ErrInvalidAuth indicates the authentication token is invalid or the user lacks permissions.
var ErrInvalidAuth = errors.New("invalid authentication")

// The WWW-Authenticate challenges that refuse a bearer request (RFC 6750 §3). A 401 must carry one
// (RFC 9110 §15.5.2), and a token that does not open the route gets one too.
const (
	// BearerChallenge answers a request without a bearer token. It names no error code (RFC 6750 §3.1).
	BearerChallenge = "Bearer"
	// BearerChallengeInvalidToken answers a token that is rejected, expired or signed out.
	BearerChallengeInvalidToken = `Bearer error="invalid_token"`
	// BearerChallengeInsufficientScope answers a valid token that lacks the permission the route requires.
	BearerChallengeInsufficientScope = `Bearer error="insufficient_scope"`
)

// AuthClaimsVerifier validates JWT access tokens and extracts claims.
type AuthClaimsVerifier interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*session.Claims, error)
}

// verifyClaimsErrors answers 401 to a token the verifier rejects, which the caller resolves by
// refreshing or signing in again. Any other failure, such as an unreachable key source, stays 500,
// so a client keeps a session it may still hold.
var verifyClaimsErrors = httpf.ErrMap{jwt.ErrTokenRejected: http.StatusUnauthorized}

// Auth provides JWT-based authentication and role-based authorization middleware.
// It verifies access tokens and checks that the user has at least one of the required permissions.
type Auth struct {
	// permissionsByRole maps a role name to every permission it grants.
	permissionsByRole map[string][]string

	claimsVerifier AuthClaimsVerifier

	logger logging.Log
}

// NewAuth returns an [Auth] that verifies access tokens with the given claims
// verifier and resolves caller roles to permissions through permissionsByRole.
func NewAuth(
	claimsVerifier AuthClaimsVerifier,
	permissionsByRole map[string][]string,
	logger logging.Log,
) *Auth {
	return &Auth{
		permissionsByRole: permissionsByRole,
		claimsVerifier:    claimsVerifier,
		logger:            logger,
	}
}

// Middleware returns an HTTP middleware that authenticates the request and gates
// it on the supplied permission set. With no required permissions, an authenticated
// user is admitted and an unauthenticated request passes through with no claims on
// the context (use this for optional-auth endpoints). With one or more required
// permissions, the request is admitted when at least one of the user's role-granted
// permissions is in the required set.
//
// Verified claims are stored on the request context for downstream handlers; use
// [session.GetClaimsContext] or [session.MustGetClaimsContext] to retrieve them.
func (middleware *Auth) Middleware(requiredPermissions []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middleware.serve(w, r, next, requiredPermissions)
		})
	}
}

// serve authenticates one request, then hands it to next with its claims on the context.
func (middleware *Auth) serve(
	w http.ResponseWriter, r *http.Request, next http.Handler, requiredPermissions []string,
) {
	ctx, span := otel.Tracer().Start(r.Context(), "middlewares.Auth")
	defer span.End()

	header := r.Header.Get("Authorization")
	// Optional-auth endpoint: pass through with no claims on the context.
	if header == "" && len(requiredPermissions) == 0 {
		next.ServeHTTP(w, r.WithContext(ctx))

		return
	}

	// The scheme matches in any case, and one or more spaces precede the token, which holds none
	// (RFC 9110 §11.1, §11.4, RFC 6750 §2.1).
	scheme, token, _ := strings.Cut(header, " ")
	token = strings.TrimLeft(token, " ")

	if !strings.EqualFold(scheme, "Bearer") || token == "" || strings.Contains(token, " ") {
		w.Header().Set("WWW-Authenticate", BearerChallenge)
		httpf.HandleError(
			ctx, middleware.logger, w, span,
			httpf.ErrMap{nil: http.StatusUnauthorized},
			fmt.Errorf("%w: no bearer token", ErrInvalidAuth),
		)

		return
	}

	verifyRequest := &servicejsonkeys.VerifyClaimsRequest{
		Usage:       servicejsonkeys.KeyUsageAuth,
		AccessToken: token,
	}

	claims, err := middleware.claimsVerifier.VerifyClaims(ctx, verifyRequest)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenRejected) {
			w.Header().Set("WWW-Authenticate", BearerChallengeInvalidToken)
		}

		httpf.HandleError(ctx, middleware.logger, w, span, verifyClaimsErrors, err)

		return
	}

	ctx = session.SetClaimsContext(ctx, claims)

	if len(requiredPermissions) > 0 {
		err = middleware.authorize(claims.Roles, requiredPermissions)
		if err != nil {
			if errors.Is(err, ErrInvalidAuth) {
				w.Header().Set("WWW-Authenticate", BearerChallengeInsufficientScope)
			}

			httpf.HandleError(ctx, middleware.logger, w, span, httpf.ErrMap{ErrInvalidAuth: http.StatusForbidden}, err)

			return
		}
	}

	next.ServeHTTP(w, r.WithContext(ctx))
}

// authorize returns nil once a role grants one of the required permissions, and [ErrInvalidAuth]
// when none does. A role the configuration does not define returns [authconfig.ErrUnknownRole]
// instead: this system signed the token, so the role reveals a data or configuration fault that
// answers 500, never a permission denial identical to a legitimate one.
func (middleware *Auth) authorize(roles, requiredPermissions []string) error {
	for _, role := range roles {
		permissions, known := middleware.permissionsByRole[role]
		if !known {
			return fmt.Errorf("%w: %q in token", authconfig.ErrUnknownRole, role)
		}

		if lo.Some(permissions, requiredPermissions) {
			return nil
		}
	}

	return fmt.Errorf("%w: user does not have any of the required permissions", ErrInvalidAuth)
}
