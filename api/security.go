package api

import (
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/samber/lo"

	"github.com/a-novel-kit/configurator"
	"github.com/a-novel-kit/context"
	sentryctx "github.com/a-novel-kit/context/sentry"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/models"
)

var (
	ErrMissingUserID  = errors.New("claims do not contain user ID")
	ErrPermission     = errors.New("user does not have the required permissions")
	ErrAuthentication = errors.New("authentication failed")
)

type SecurityHandlerService interface {
	Authenticate(ctx context.Context, accessToken string) (*models.AccessTokenClaims, error)
}

type SecurityHandler struct {
	// Permissions required by each operation.
	RequiredPermissions map[codegen.OperationName][]models.Permission
	// Permissions granted by each role.
	GrantedPermissions map[models.Role][]models.Permission

	SecurityHandlerService SecurityHandlerService
}

func (security *SecurityHandler) HandleBearerAuth(
	ctx context.Context, operationName codegen.OperationName, auth codegen.BearerAuth,
) (context.Context, error) {
	logger := zerolog.Ctx(ctx)

	// Get the claims from the token. This will also perform the necessary checks to ensure the token is valid.
	claims, err := security.SecurityHandlerService.Authenticate(ctx, auth.Token)
	if err != nil {
		return nil, errors.Join(err, ErrAuthentication)
	}

	sentryctx.ConfigureScope(ctx, func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID: lo.FromPtr(claims.UserID).String(),
		})
	})

	// List the permissions required by the current operation.
	requiredPermissions, ok := security.RequiredPermissions[operationName]
	if !ok {
		return context.WithValue(ctx, ClaimsAPIKey{}, claims), nil
	}

	// Retrieve all the permissions granted to the user, based on its roles.
	var grantedPermissions []models.Permission

	for _, role := range claims.Roles {
		if rolePermissions, ok := security.GrantedPermissions[role]; ok {
			grantedPermissions = append(grantedPermissions, rolePermissions...)
		}
	}

	// Check the intersection between the 2 sets of permissions.
	matchedPermissions := lo.Intersect(requiredPermissions, grantedPermissions)

	sentryctx.AddBreadcrumb(ctx, &sentry.Breadcrumb{
		Category: "permissions",
		Message:  "check permissions",
		Data: map[string]any{
			"required_permissions": requiredPermissions,
			"claims_permissions":   grantedPermissions,
		},
		Level:     sentry.LevelInfo,
		Timestamp: time.Now(),
	}, nil)

	// If every required permission matched, then the intersection with the claims permissions
	// is the same length as the required permissions.
	if len(matchedPermissions) != len(requiredPermissions) {
		logger.
			Warn().
			Err(models.ErrUnauthorized).
			Any("required_permissions", requiredPermissions).
			Any("claims_permissions", grantedPermissions).
			Msg("check permissions")

		return nil, ErrPermission
	}

	return context.WithValue(ctx, ClaimsAPIKey{}, claims), nil
}

func NewSecurity(
	required map[codegen.OperationName][]models.Permission,
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
		RequiredPermissions:    required,
		GrantedPermissions:     resolveGranted,
		SecurityHandlerService: service,
	}, nil
}

type ClaimsAPIKey struct{}

func GetSecurityClaims(ctx context.Context) (*models.AccessTokenClaims, error) {
	claims, err := context.ExtractValue[*models.AccessTokenClaims](ctx, ClaimsAPIKey{})
	if err != nil {
		return nil, fmt.Errorf("(GetSecurityClaims) extract claims: %w", err)
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
