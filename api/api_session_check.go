package api

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/models"
)

func (api *API) CheckSession(ctx context.Context) (codegen.CheckSessionRes, error) {
	// Extract the claims from the context. This should never fail, as this handler is only triggered after a
	// successful authentication.
	claims, err := context.ExtractValue[*models.AccessTokenClaims](ctx, ClaimsAPIKey{})
	if err != nil {
		return nil, fmt.Errorf("extract claims: %w", err)
	}

	return &codegen.Claims{
		UserID: codegen.OptUUID{Value: lo.FromPtr(claims.UserID), Set: claims.UserID != nil},
		Roles: lo.Map(claims.Roles, func(item models.Role, _ int) codegen.Role {
			return codegen.Role(item)
		}),
		RefreshTokenID: codegen.OptString{Value: lo.FromPtr(claims.RefreshTokenID), Set: claims.RefreshTokenID != nil},
	}, nil
}
