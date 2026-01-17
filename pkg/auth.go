package pkg

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/a-novel-kit/golib/deps"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/services"
)

type Role = config.Role

type Permissions = config.Permissions

type Claims = services.AccessTokenClaims

type PermissionsHandler func(r chi.Router, permissions ...string) chi.Router

func NewAuthHandler(
	claimsVerifier middlewares.AuthClaimsVerifier,
	permissions Permissions,
) PermissionsHandler {
	permissionsByRole := lo.Must(deps.ResolveDependants[string, string](
		lo.MapEntries(
			permissions.Roles,
			func(key string, value config.Role) (string, []string) {
				return key, value.Permissions
			},
		),
		lo.MapEntries(permissions.Roles, func(key string, value config.Role) (string, []string) {
			return key, value.Inherits
		}),
	))

	middlewareAuth := middlewares.NewAuth(claimsVerifier, permissionsByRole)

	return func(r chi.Router, permissions ...string) chi.Router {
		return r.With(middlewareAuth.Middleware(permissions))
	}
}

func SetClaimsContext(ctx context.Context, claims *Claims) context.Context {
	return middlewares.SetClaimsContext(ctx, claims)
}

func GetClaimsContext(ctx context.Context) (*Claims, error) {
	return middlewares.GetClaimsContext(ctx)
}

func MustGetClaimsContext(ctx context.Context) (*Claims, error) {
	return middlewares.MustGetClaimsContext(ctx)
}
