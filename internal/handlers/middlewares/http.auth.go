package middlewares

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/a-novel/golib/httpf"
	"github.com/a-novel/golib/otel"
	jkmodels "github.com/a-novel/service-json-keys/models"
	jkpkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/services"
)

var (
	ErrMissingAuth = errors.New("missing auth")
	ErrInvalidAuth = errors.New("invalid authentication")
)

type AuthClaimsVerifier interface {
	VerifyClaims(
		ctx context.Context, usage jkmodels.KeyUsage, accessToken string, options *jkpkg.VerifyClaimsOptions,
	) (*services.AccessTokenClaims, error)
}

type Auth struct {
	permissionsByRole map[string][]string

	claimsVerifier AuthClaimsVerifier
}

func NewAuth(
	claimsVerifier AuthClaimsVerifier,
	permissionsByRole map[string][]string,
) *Auth {
	return &Auth{
		permissionsByRole: permissionsByRole,
		claimsVerifier:    claimsVerifier,
	}
}

func (middleware *Auth) Middleware(requiredPermissions []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := otel.Tracer().Start(r.Context(), "middlewares.Auth")
			defer span.End()

			token := r.Header.Get("Authorization")
			// No token but no permissions required.
			if token == "" && len(requiredPermissions) == 0 {
				otel.ReportSuccessNoContent(span)
				next.ServeHTTP(w, r.WithContext(ctx))

				return
			}

			if token == "" {
				httpf.HandleError(
					ctx, w, span,
					httpf.ErrMap{nil: http.StatusUnauthorized},
					fmt.Errorf("%w: missing authorization header", ErrInvalidAuth),
				)

				return
			}

			authToken := strings.Split(token, " ")
			if len(authToken) != 2 || authToken[0] != "Bearer" {
				httpf.HandleError(
					ctx, w, span,
					httpf.ErrMap{nil: http.StatusUnauthorized},
					fmt.Errorf("%w: invalid authorization header", ErrInvalidAuth),
				)

				return
			}

			accessToken := authToken[1]

			claims, err := middleware.claimsVerifier.VerifyClaims(ctx, jkmodels.KeyUsageAuth, accessToken, nil)
			if err != nil {
				httpf.HandleError(ctx, w, span, map[error]int{jws.ErrInvalidSignature: http.StatusUnauthorized}, err)

				return
			}

			ctx = SetClaimsContext(ctx, claims)

			// Assign an object with each required permissions, then toggle the permission flag if the user has it.
			grantedPermissions := map[string]bool{}

			for _, permission := range requiredPermissions {
				grantedPermissions[permission] = false
			}

			for _, role := range claims.Roles {
				for _, permission := range middleware.permissionsByRole[role] {
					grantedPermissions[permission] = true
				}
			}

			// If a permission remains with status false, it means the user lacks it so we refuse access.
			for permission, granted := range grantedPermissions {
				if !granted {
					httpf.HandleError(
						ctx, w, span,
						httpf.ErrMap{nil: http.StatusForbidden},
						fmt.Errorf("%w: user needs '%s' permission", ErrInvalidAuth, permission),
					)

					return
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
			otel.ReportSuccessNoContent(span)
		})
	}
}

type ClaimsContextKey struct{}

func SetClaimsContext(ctx context.Context, claims *services.AccessTokenClaims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey{}, claims)
}

func GetClaimsContext(ctx context.Context) (*services.AccessTokenClaims, error) {
	claims, ok := ctx.Value(ClaimsContextKey{}).(*services.AccessTokenClaims)
	if !ok && claims != nil {
		return nil, fmt.Errorf(
			"(GetClaimsContext) extract claims: got type %T, expected %T",
			ctx.Value(ClaimsContextKey{}), &services.AccessTokenClaims{},
		)
	}

	return claims, nil
}

func MustGetClaimsContext(ctx context.Context) (*services.AccessTokenClaims, error) {
	claims, err := GetClaimsContext(ctx)
	if err != nil {
		return nil, err
	}

	if claims == nil {
		return nil, ErrMissingAuth
	}

	return claims, nil
}
