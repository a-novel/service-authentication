package services

import (
	"context"
	"fmt"

	"github.com/a-novel/golib/otel"
	jkmodels "github.com/a-novel/service-json-keys/models"

	"github.com/a-novel/service-authentication/models"
)

// LoginAnonSource is the source used to perform the LoginAnonService.LoginAnon action.
type LoginAnonSource interface {
	SignClaims(ctx context.Context, usage jkmodels.KeyUsage, claims any) (string, error)
}

// LoginAnonService is the service used to perform the LoginAnonService.LoginAnon action.
//
// You may create one using the NewLoginAnonService function.
type LoginAnonService struct {
	source LoginAnonSource
}

func NewLoginAnonService(source LoginAnonSource) *LoginAnonService {
	return &LoginAnonService{source: source}
}

// LoginAnon creates a new anonymous session. Anonymous sessions grant basic access to public protected resources.
//
// On success, a new access token is returned, so the user can access protected resources.
func (service *LoginAnonService) LoginAnon(ctx context.Context) (string, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.LoginAnon")
	defer span.End()

	accessToken, err := service.source.SignClaims(
		ctx,
		jkmodels.KeyUsageAuth,
		models.AccessTokenClaims{
			Roles: []models.Role{models.RoleAnon},
		},
	)
	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("issue accessToken: %w", err))
	}

	return otel.ReportSuccess(span, accessToken), nil
}
