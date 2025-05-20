package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var ErrLoginService = errors.New("LoginService.Login")

func NewErrLoginService(err error) error {
	return errors.Join(err, ErrLoginService)
}

// LoginSource is the source used to perform the LoginService.Login action.
//
// You may build one using the NewLoginServiceSource function.
type LoginSource interface {
	SelectCredentialsByEmail(ctx context.Context, email string) (*dao.CredentialsEntity, error)
	IssueToken(ctx context.Context, request IssueTokenRequest) (string, error)
}

// LoginRequest is the request sent by the client to log in.
type LoginRequest struct {
	// Email of the user trying to log in.
	Email string
	// Password of the user trying to log in.
	Password string
}

// LoginService is the service used to perform the LoginService.Login action.
//
// You may create one using the NewLoginService function.
type LoginService struct {
	source LoginSource
}

// Login a user.
//
// On success, a new access token is returned, so the user can access protected resources.
//
// You may also create an anonymous session using the LoginAnonService.
func (service *LoginService) Login(ctx context.Context, request LoginRequest) (string, error) {
	// Retrieve credentials.
	credentials, err := service.source.SelectCredentialsByEmail(ctx, request.Email)
	if err != nil {
		return "", NewErrLoginService(fmt.Errorf("select credentials by email: %w", err))
	}

	// Validate password.
	if err = lib.CompareScrypt(request.Password, credentials.Password); err != nil {
		return "", NewErrLoginService(fmt.Errorf("compare password: %w", err))
	}

	// Generate a new authentication token.
	accessToken, err := service.source.IssueToken(ctx, IssueTokenRequest{
		UserID: &credentials.ID,
		Roles: []models.Role{
			lo.Switch[models.CredentialsRole, models.Role](credentials.Role).
				Case(models.CredentialsRoleAdmin, models.RoleAdmin).
				Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
				Default(models.RoleUser),
		},
	})
	if err != nil {
		return "", NewErrLoginService(fmt.Errorf("issue accessToken: %w", err))
	}

	return accessToken, nil
}

func NewLoginServiceSource(
	selectCredentialsByEmailDAO *dao.SelectCredentialsByEmailRepository,
	issueTokenService *IssueTokenService,
) LoginSource {
	return &struct {
		*dao.SelectCredentialsByEmailRepository
		*IssueTokenService
	}{
		SelectCredentialsByEmailRepository: selectCredentialsByEmailDAO,
		IssueTokenService:                  issueTokenService,
	}
}

func NewLoginService(source LoginSource) *LoginService {
	return &LoginService{source: source}
}
