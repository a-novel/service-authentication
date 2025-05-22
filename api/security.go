package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/samber/lo"

	"github.com/a-novel-kit/configurator"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/models"
)

var (
	ErrMissingUserID  = errors.New("claims do not contain user ID")
	ErrPermission     = errors.New("user does not have the required permissions")
	ErrAuthentication = errors.New("authentication failed")
	ErrInvalidClaims  = errors.New("invalid claims")
)

type SecurityHandlerService interface {
	Authenticate(ctx context.Context, accessToken string) (*models.AccessTokenClaims, error)
}

type SecurityHandler struct {
	// Permissions granted by each role.
	GrantedPermissions map[models.Role][]models.Permission

	SecurityHandlerService SecurityHandlerService
}

func (security *SecurityHandler) HandleBearerAuth(
	ctx context.Context, operationName codegen.OperationName, auth codegen.BearerAuth,
) (context.Context, error) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("operation", operationName).
		Logger()

	// Get the claims from the token. This will also perform the necessary checks to ensure the token is valid.
	claims, err := security.SecurityHandlerService.Authenticate(ctx, auth.Token)
	if err != nil {
		return nil, errors.Join(err, ErrAuthentication)
	}

	hub := sentry.GetHubFromContext(ctx)
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
		if rolePermissions, ok := security.GrantedPermissions[role]; ok {
			for _, permission := range rolePermissions {
				grantedPermissions[permission] = struct{}{}
			}
		}
	}

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: "permissions",
		Message:  "check permissions",
		Data: map[string]any{
			"required_permissions": auth.Roles,
			"claims_permissions":   grantedPermissions,
		},
		Level:     sentry.LevelInfo,
		Timestamp: time.Now(),
	}, nil)

	// Check if the user has the required permissions for the operation.
	for _, permission := range auth.Roles {
		if _, ok := grantedPermissions[models.Permission(permission)]; !ok {
			logger.
				Warn().
				Err(models.ErrUnauthorized).
				Any("required_permissions", auth.Roles).
				Any("claims_permissions", grantedPermissions).
				Msg("check permissions")

			return nil, ErrPermission
		}
	}

	return context.WithValue(ctx, ClaimsAPIKey{}, claims), nil
}

func NewSecurity(
	granted models.PermissionsConfig,
	service SecurityHandlerService,
) (*SecurityHandler, error) {
	resolveGranted, err := configurator.ResolveDependants[models.Role, models.Permission](
		lo.MapEntries(
			granted.Roles,
			func(key models.Role, value models.RoleConfig) (models.Role, []models.Permission) {
				return key, value.Permissions
			},
		),
		lo.MapEntries(granted.Roles, func(key models.Role, value models.RoleConfig) (models.Role, []models.Role) {
			return key, value.Inherits
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("(NewSecurity) resolve granted permissions: %w", err)
	}

	return &SecurityHandler{
		GrantedPermissions:     resolveGranted,
		SecurityHandlerService: service,
	}, nil
}

type ClaimsAPIKey struct{}

func GetSecurityClaims(ctx context.Context) (*models.AccessTokenClaims, error) {
	claims, ok := ctx.Value(ClaimsAPIKey{}).(*models.AccessTokenClaims)
	if !ok {
		return nil, fmt.Errorf(
			"(GetSecurityClaims) extract claims: %w: got type %T, expected %T",
			ErrInvalidClaims,
			ctx.Value(ClaimsAPIKey{}), &models.AccessTokenClaims{},
		)
	}

	return claims, nil
}

func RequireUserID(ctx context.Context) (uuid.UUID, error) {
	claims, err := GetSecurityClaims(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("(RequireUserID): %w", err)
	}

	if claims.UserID == nil {
		return uuid.Nil, ErrMissingUserID
	}

	return *claims.UserID, nil
}
