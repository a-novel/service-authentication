package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	jkModels "github.com/a-novel/service-json-keys/models"

	"github.com/a-novel/service-authentication/models"
)

var ErrLoginAnonService = errors.New("LoginAnonService.LoginAnon")

func NewErrLoginAnonService(err error) error {
	return errors.Join(err, ErrLoginAnonService)
}

// LoginAnonSource is the source used to perform the LoginAnonService.LoginAnon action.
type LoginAnonSource interface {
	SignClaims(ctx context.Context, usage jkModels.KeyUsage, claims any) (string, error)
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
	span := sentry.StartSpan(ctx, "LoginAnonService.LoginAnon")
	defer span.Finish()

	accessToken, err := service.source.SignClaims(
		span.Context(),
		jkModels.KeyUsageAuth,
		models.AccessTokenClaims{
			Roles: []models.Role{models.RoleAnon},
		},
	)
	if err != nil {
		span.SetData("service.error", err.Error())

		return "", NewErrLoginAnonService(fmt.Errorf("issue accessToken: %w", err))
	}

	return accessToken, nil
}
