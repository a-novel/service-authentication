package api

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/samber/lo"

	"github.com/a-novel-kit/configurator"
	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/models"
)

var ErrMissingUserID = errors.New("claims do not contain user ID")

type SecurityAPISource interface {
	Authenticate(ctx context.Context, accessToken string) (*models.AccessTokenClaims, error)
}

type SecurityAPI struct {
	// Permissions required by each operation.
	RequiredPermissions map[codegen.OperationName][]models.Permission
	// Permissions granted by each role.
	GrantedPermissions map[models.Role][]models.Permission

	SecurityAPISource SecurityAPISource
}

func (security *SecurityAPI) HandleBearerAuth(
	ctx context.Context, operationName codegen.OperationName, auth codegen.BearerAuth,
) (context.Context, error) {
	logger := zerolog.Ctx(ctx)

	// Get the claims from the token. This will also perform the necessary checks to ensure the token is valid.
	claims, err := security.SecurityAPISource.Authenticate(ctx, auth.Token)
	if err != nil {
		logger.Warn().Err(err).Msg("authenticate user")

		return nil, fmt.Errorf("authenticate user: %w", err)
	}

	// List the permissions required by the current operation.
	requiredPermissions, ok := security.RequiredPermissions[operationName]
	if !ok {
		return context.WithValue(ctx, claimsAPIKey{}, claims), nil
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

	// If every required permission matched, then the intersection with the claims permissions
	// is the same length as the required permissions.
	if len(matchedPermissions) != len(requiredPermissions) {
		logger.
			Warn().
			Err(models.ErrUnauthorized).
			Any("required_permissions", requiredPermissions).
			Any("claims_permissions", grantedPermissions).
			Msg("check permissions")

		return nil, models.ErrUnauthorized
	}

	return context.WithValue(ctx, claimsAPIKey{}, claims), nil
}

func NewSecurity(
	required map[codegen.OperationName][]models.Permission,
	granted models.PermissionsConfig,
	service SecurityAPISource,
) (*SecurityAPI, error) {
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

	return &SecurityAPI{
		RequiredPermissions: required,
		GrantedPermissions:  resolveGranted,
		SecurityAPISource:   service,
	}, nil
}

type claimsAPIKey struct{}

func GetSecurityClaims(ctx context.Context) (*models.AccessTokenClaims, error) {
	claims, err := context.ExtractValue[*models.AccessTokenClaims](ctx, claimsAPIKey{})
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
