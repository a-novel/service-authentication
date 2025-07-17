package pkg

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	jkModels "github.com/a-novel/service-json-keys/models"
	jkPkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/configurator"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/config"
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
	source AuthenticateSource, permissions config.Permissions,
) (*HandleBearerAuth[OpName], error) {
	resolveGranted, err := configurator.ResolveDependants[models.Role, models.Permission](
		lo.MapEntries(
			permissions.Roles,
			func(key models.Role, value config.Role) (models.Role, []models.Permission) {
				return key, value.Permissions
			},
		),
		lo.MapEntries(permissions.Roles, func(key models.Role, value config.Role) (models.Role, []models.Role) {
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
	ctx, span := otel.Tracer().Start(ctx, "pkg.HandleBearerAuth")
	defer span.End()

	span.SetAttributes(attribute.String("request.operation", string(operationName)))

	// If no token is provided, we cannot authenticate.
	if auth.GetToken() == "" {
		return ctx, otel.ReportError(span, fmt.Errorf("token is empty: %w", models.ErrUnauthorized))
	}

	claims, err := handler.source.VerifyClaims(ctx, jkModels.KeyUsageAuth, auth.GetToken(), nil)
	if err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) {
			return ctx, otel.ReportError(span, fmt.Errorf("consume token: %w", models.ErrUnauthorized))
		}

		return ctx, otel.ReportError(span, fmt.Errorf("verify claims: %w", err))
	}

	// Retrieve all the permissions granted to the user, based on its roles.
	grantedPermissions := make(map[models.Permission]struct{})

	for _, role := range claims.Roles {
		if rolePermissions, ok := handler.permissions[role]; ok {
			for _, permission := range rolePermissions {
				grantedPermissions[permission] = struct{}{}
			}
		}
	}

	// Check if the user has the required permissions for the operation.
	for _, permission := range auth.GetRoles() {
		if _, ok := grantedPermissions[models.Permission(permission)]; !ok {
			return ctx, otel.ReportError(span, fmt.Errorf("%w: missing permission %s", models.ErrForbidden, permission))
		}
	}

	// Set the claims in the context for further use.
	ctx = SetClaimsContext(ctx, claims)

	return otel.ReportSuccess(span, ctx), nil
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
