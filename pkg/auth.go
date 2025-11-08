package pkg

import (
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/a-novel/golib/deps"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/handlers/middlewares"
)

type Permissions = config.Permissions

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
