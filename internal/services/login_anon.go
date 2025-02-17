package services

import (
	"fmt"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/models"
)

// LoginAnonSource is the source used to perform the LoginAnonService.LoginAnon action.
type LoginAnonSource interface {
	IssueToken(ctx context.Context, request IssueTokenRequest) (string, error)
}

// LoginAnonService is the service used to perform the LoginAnonService.LoginAnon action.
//
// You may create one using the NewLoginAnonService function.
type LoginAnonService struct {
	source LoginAnonSource
}

// LoginAnon creates a new anonymous session. Anonymous sessions grant basic access to public protected resources.
//
// On success, a new access token is returned, so the user can access protected resources.
func (service *LoginAnonService) LoginAnon(ctx context.Context) (string, error) {
	accessToken, err := service.source.IssueToken(ctx, IssueTokenRequest{
		Roles: []models.Role{models.RoleAnon},
	})
	if err != nil {
		return "", fmt.Errorf("(LoginAnonService.LoginAnon) issue accessToken: %w", err)
	}

	return accessToken, nil
}

func NewLoginAnonService(source LoginAnonSource) *LoginAnonService {
	return &LoginAnonService{source: source}
}
