package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

type ConsumeRefreshTokenService interface {
	ConsumeRefreshToken(ctx context.Context, request services.ConsumeRefreshTokenRequest) (string, error)
}

func (api *API) RefreshSession(
	ctx context.Context, params codegen.RefreshSessionParams,
) (codegen.RefreshSessionRes, error) {
	span := sentry.StartSpan(ctx, "API.RefreshSession")
	defer span.Finish()

	accessToken, err := api.ConsumeRefreshTokenService.ConsumeRefreshToken(
		span.Context(),
		services.ConsumeRefreshTokenRequest{
			AccessToken:  params.AccessToken,
			RefreshToken: params.RefreshToken,
		},
	)

	switch {
	case errors.Is(err, models.ErrUnauthorized):
		span.SetData("service.err", err.Error())

		return &codegen.ForbiddenError{Error: "invalid user password"}, nil
	case errors.Is(err, services.ErrMismatchRefreshClaims),
		errors.Is(err, services.ErrTokenIssuedWithDifferentRefreshToken):
		span.SetData("service.err", err.Error())

		return &codegen.UnprocessableEntityError{Error: "invalid refresh token"}, nil
	case err != nil:
		span.SetData("service.err", err.Error())

		return nil, fmt.Errorf("refresh session: %w", err)
	}

	return &codegen.Token{AccessToken: accessToken}, nil
}
