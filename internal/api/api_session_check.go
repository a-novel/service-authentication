package api

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

func (api *API) CheckSession(ctx context.Context) (apimodels.CheckSessionRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.CheckSession")
	defer span.End()

	// Extract the claims from the context. This should never fail, as this handler is only triggered after a
	// successful authentication.
	claims, err := pkg.GetClaimsContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get claims: %w", err))
	}

	return otel.ReportSuccess(span, &apimodels.Claims{
		UserID: apimodels.OptUUID{Value: lo.FromPtr(claims.UserID), Set: claims.UserID != nil},
		Roles: lo.Map(claims.Roles, func(item models.Role, _ int) apimodels.Role {
			return apimodels.Role(item)
		}),
		RefreshTokenID: apimodels.OptString{Value: lo.FromPtr(claims.RefreshTokenID), Set: claims.RefreshTokenID != nil},
	}), nil
}
