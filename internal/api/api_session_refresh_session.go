package api

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel/golib/otel"

	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

type ConsumeRefreshTokenService interface {
	ConsumeRefreshToken(ctx context.Context, request services.ConsumeRefreshTokenRequest) (string, error)
}

func (api *API) RefreshSession(
	ctx context.Context, params apimodels.RefreshSessionParams,
) (apimodels.RefreshSessionRes, error) {
	ctx, span := otel.Tracer().Start(ctx, "api.RefreshSession")
	defer span.End()

	accessToken, err := api.ConsumeRefreshTokenService.ConsumeRefreshToken(
		ctx,
		services.ConsumeRefreshTokenRequest{
			AccessToken:  params.AccessToken,
			RefreshToken: params.RefreshToken,
		},
	)

	switch {
	case errors.Is(err, models.ErrUnauthorized):
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return &apimodels.ForbiddenError{Error: "invalid user password"}, nil
	case errors.Is(err, services.ErrMismatchRefreshClaims),
		errors.Is(err, services.ErrTokenIssuedWithDifferentRefreshToken):
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return &apimodels.UnprocessableEntityError{Error: "invalid refresh token"}, nil
	case err != nil:
		span.RecordError(err)
		span.SetStatus(codes.Error, "")

		return nil, fmt.Errorf("refresh session: %w", err)
	}

	return otel.ReportSuccess(span, &apimodels.Token{
		AccessToken:  accessToken,
		RefreshToken: params.RefreshToken,
	}), nil
}
