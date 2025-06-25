package api

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"

	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/models"
)

func (api *API) CheckSession(ctx context.Context) (codegen.CheckSessionRes, error) {
	span := sentry.StartSpan(ctx, "API.CheckSession")
	defer span.Finish()

	// Extract the claims from the context. This should never fail, as this handler is only triggered after a
	// successful authentication.
	claims, err := GetSecurityClaims(span.Context())
	if err != nil {
		span.SetData("session.err", err.Error())

		return nil, fmt.Errorf("get claims: %w", err)
	}

	span.SetData("session.userID", lo.FromPtr(claims.UserID))
	span.SetData("session.roles", claims.Roles)
	span.SetData("session.refreshTokenID", lo.FromPtr(claims.RefreshTokenID))

	return &codegen.Claims{
		UserID: codegen.OptUUID{Value: lo.FromPtr(claims.UserID), Set: claims.UserID != nil},
		Roles: lo.Map(claims.Roles, func(item models.Role, _ int) codegen.Role {
			return codegen.Role(item)
		}),
		RefreshTokenID: codegen.OptString{Value: lo.FromPtr(claims.RefreshTokenID), Set: claims.RefreshTokenID != nil},
	}, nil
}
