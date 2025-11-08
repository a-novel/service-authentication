package services

import (
	"context"
	"fmt"

	"github.com/a-novel/golib/otel"
	jkmodels "github.com/a-novel/service-json-keys/models"

	"github.com/a-novel/service-authentication/internal/config"
)

type TokenCreateAnonSignClaimsService interface {
	SignClaims(ctx context.Context, usage jkmodels.KeyUsage, claims any) (string, error)
}

type TokenCreateAnon struct {
	signClaimsService TokenCreateServiceSignClaims
}

func NewTokenCreateAnon(signClaimsService TokenCreateServiceSignClaims) *TokenCreateAnon {
	return &TokenCreateAnon{
		signClaimsService: signClaimsService,
	}
}

func (service *TokenCreateAnon) Exec(ctx context.Context) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.TokenCreateAnon")
	defer span.End()

	accessToken, err := service.signClaimsService.SignClaims(
		ctx,
		jkmodels.KeyUsageAuth,
		AccessTokenClaims{
			Roles: []string{config.RoleAnon},
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue accessToken: %w", err))
	}

	return otel.ReportSuccess(span, &Token{
		AccessToken: accessToken,
	}), nil
}
