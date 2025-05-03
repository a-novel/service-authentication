package api

import (
	"errors"
	"fmt"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/services"
)

type IssueRefreshTokenService interface {
	IssueRefreshToken(ctx context.Context, request services.IssueRefreshTokenRequest) (string, error)
}

func (api *API) CreateRefreshToken(ctx context.Context) (codegen.CreateRefreshTokenRes, error) {
	claims, err := GetSecurityClaims(ctx)
	if err != nil {
		return nil, fmt.Errorf("get security claims: %w", err)
	}

	refreshToken, err := api.IssueRefreshTokenService.IssueRefreshToken(
		ctx,
		services.IssueRefreshTokenRequest{Claims: claims},
	)

	switch {
	case errors.Is(err, services.ErrRefreshRefreshToken),
		errors.Is(err, services.ErrRefreshTokenWithAnonSession):
		return &codegen.ForbiddenError{Error: "invalid access token"}, nil
	case err != nil:
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	return &codegen.RefreshToken{RefreshToken: refreshToken}, nil
}
