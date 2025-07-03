package pkg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/samber/lo"

	jkModels "github.com/a-novel/service-json-keys/models"
	jkPkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/configurator"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/models"
)

type AuthenticateSource interface {
	VerifyClaims(
		ctx context.Context, usage jkModels.KeyUsage, accessToken string, options *jkPkg.VerifyClaimsOptions,
	) (*models.AccessTokenClaims, error)
}

type Token interface {
	GetToken() string
	GetRoles() []string
}

type ClaimsContextKey struct{}

type HandleBearerAuth[OpName string] struct {
	source      AuthenticateSource
	permissions map[models.Role][]models.Permission
}

func NewHandleBearerAuth[OpName string](
	source AuthenticateSource, permissions models.PermissionsConfig,
) (*HandleBearerAuth[OpName], error) {
	resolveGranted, err := configurator.ResolveDependants[models.Role, models.Permission](
		lo.MapEntries(
			permissions.Roles,
			func(key models.Role, value models.RoleConfig) (models.Role, []models.Permission) {
				return key, value.Permissions
			},
		),
		lo.MapEntries(permissions.Roles, func(key models.Role, value models.RoleConfig) (models.Role, []models.Role) {
			return key, value.Inherits
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("(NewSecurity) resolve granted permissions: %w", err)
	}

	return &HandleBearerAuth[OpName]{source: source, permissions: resolveGranted}, nil
}

func (handler *HandleBearerAuth[OpName]) HandleBearerAuth(
	ctx context.Context, operationName OpName, auth Token,
) (context.Context, error) {
	span := sentry.StartSpan(ctx, "Pkg.HandleBearerAuth")
	defer span.Finish()

	span.SetData("request.operation", operationName)

	// If no token is provided, we cannot authenticate.
	if auth.GetToken() == "" {
		span.SetData("authenticate.err", "token is empty")

		return ctx, fmt.Errorf("token is empty: %w", models.ErrUnauthorized)
	}

	claims, err := handler.source.VerifyClaims(span.Context(), jkModels.KeyUsageAuth, auth.GetToken(), nil)
	if err != nil {
		span.SetData("error", err.Error())
		sentry.CaptureException(err)

		if errors.Is(err, jws.ErrInvalidSignature) {
			return ctx, fmt.Errorf("consume token: %w", models.ErrUnauthorized)
		}

		return ctx, fmt.Errorf("verify claims: %w", err)
	}

	hub := sentry.GetHubFromContext(span.Context())
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID: lo.FromPtr(claims.UserID).String(),
		})
	})

	// Retrieve all the permissions granted to the user, based on its roles.
	grantedPermissions := make(map[models.Permission]struct{})

	for _, role := range claims.Roles {
		if rolePermissions, ok := handler.permissions[role]; ok {
			for _, permission := range rolePermissions {
				grantedPermissions[permission] = struct{}{}
			}
		}
	}

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "permissions",
		Message:  "check permissions",
		Data: map[string]any{
			"required_permissions": auth.GetRoles(),
			"claims_permissions":   grantedPermissions,
		},
		Level:     sentry.LevelInfo,
		Timestamp: time.Now(),
	}, nil)

	// Check if the user has the required permissions for the operation.
	for _, permission := range auth.GetRoles() {
		if _, ok := grantedPermissions[models.Permission(permission)]; !ok {
			span.SetData("permission.err", "missing permission: "+permission)

			return ctx, fmt.Errorf("%w: missing permission %s", models.ErrForbidden, permission)
		}
	}

	// Set the claims in the context for further use.
	ctx = SetClaimsContext(span.Context(), claims)
	span.SetData("claims", claims)

	return ctx, nil
}

func SetClaimsContext(ctx context.Context, claims *models.AccessTokenClaims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey{}, claims)
}

func GetClaimsContext(ctx context.Context) (*models.AccessTokenClaims, error) {
	claims, ok := ctx.Value(ClaimsContextKey{}).(*models.AccessTokenClaims)
	if !ok {
		return nil, fmt.Errorf(
			"(GetClaimsContext) extract claims: got type %T, expected %T",
			ctx.Value(ClaimsContextKey{}), &models.AccessTokenClaims{},
		)
	}

	return claims, nil
}

func RequireUserID(ctx context.Context) (uuid.UUID, error) {
	claims, err := GetClaimsContext(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("(RequireUserID): %w", err)
	}

	if claims.UserID == nil {
		return uuid.Nil, fmt.Errorf("(RequireUserID): %w", models.ErrUnauthorized)
	}

	return *claims.UserID, nil
}
