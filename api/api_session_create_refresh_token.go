package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
)

type IssueRefreshTokenService interface {
	IssueRefreshToken(ctx context.Context, request services.IssueRefreshTokenRequest) (string, error)
}

func (api *API) CreateRefreshToken(ctx context.Context) (codegen.CreateRefreshTokenRes, error) {
	span := sentry.StartSpan(ctx, "API.CreateRefreshToken")
	defer span.Finish()

	claims, err := GetSecurityClaims(span.Context())
	if err != nil {
		span.SetData("claims.err", err.Error())

		return nil, fmt.Errorf("get security claims: %w", err)
	}

	span.SetData("claims.userID", claims.UserID)
	span.SetData("claims.roles", claims.Roles)
	span.SetData("session.refreshTokenID", lo.FromPtr(claims.RefreshTokenID))

	refreshToken, err := api.IssueRefreshTokenService.IssueRefreshToken(
		span.Context(),
		services.IssueRefreshTokenRequest{Claims: claims},
	)

	switch {
	case errors.Is(err, services.ErrRefreshRefreshToken),
		errors.Is(err, services.ErrRefreshTokenWithAnonSession):
		span.SetData("claims.err", err.Error())

		return &codegen.ForbiddenError{Error: "invalid access token"}, nil
	case err != nil:
		span.SetData("claims.err", err.Error())

		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	return &codegen.RefreshToken{RefreshToken: refreshToken}, nil
}
