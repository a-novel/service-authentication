package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/samber/lo"

	"github.com/a-novel-kit/jwt/jwa"

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
	IssueRefreshToken(ctx context.Context, request IssueRefreshTokenRequest) (string, *jwa.Claims, error)
}

func NewLoginServiceSource(
	selectCredentialsByEmailDAO *dao.SelectCredentialsByEmailRepository,
	issueTokenService *IssueTokenService,
	issueRefreshTokenService *IssueRefreshTokenService,
) LoginSource {
	return &struct {
		*dao.SelectCredentialsByEmailRepository
		*IssueTokenService
		*IssueRefreshTokenService
	}{
		SelectCredentialsByEmailRepository: selectCredentialsByEmailDAO,
		IssueTokenService:                  issueTokenService,
		IssueRefreshTokenService:           issueRefreshTokenService,
	}
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

func NewLoginService(source LoginSource) *LoginService {
	return &LoginService{source: source}
}

// Login a user.
//
// On success, a new access token is returned, so the user can access protected resources.
//
// You may also create an anonymous session using the LoginAnonService.
func (service *LoginService) Login(ctx context.Context, request LoginRequest) (*models.Token, error) {
	span := sentry.StartSpan(ctx, "LoginService.Login")
	defer span.Finish()

	span.SetData("email", request.Email)

	// Retrieve credentials.
	credentials, err := service.source.SelectCredentialsByEmail(span.Context(), request.Email)
	if err != nil {
		span.SetData("dao.error", err.Error())

		return nil, NewErrLoginService(fmt.Errorf("select credentials by email: %w", err))
	}

	span.SetData("userID", credentials.ID)
	span.SetData("role", credentials.Role)

	// Validate password.
	err = lib.CompareScrypt(request.Password, credentials.Password)
	if err != nil {
		span.SetData("password.compare.error", err.Error())

		return nil, NewErrLoginService(fmt.Errorf("compare password: %w", err))
	}

	refreshToken, refreshTokenClaims, err := service.source.IssueRefreshToken(span.Context(), IssueRefreshTokenRequest{
		Claims: &models.AccessTokenClaims{
			UserID: &credentials.ID,
			Roles: []models.Role{
				lo.Switch[models.CredentialsRole, models.Role](credentials.Role).
					Case(models.CredentialsRoleAdmin, models.RoleAdmin).
					Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
					Default(models.RoleUser),
			},
		},
	})
	if err != nil {
		span.SetData("issueRefreshToken.error", err.Error())

		return nil, NewErrLoginService(fmt.Errorf("issue refresh token: %w", err))
	}

	// Generate a new authentication token.
	accessToken, err := service.source.IssueToken(span.Context(), IssueTokenRequest{
		UserID: &credentials.ID,
		Roles: []models.Role{
			lo.Switch[models.CredentialsRole, models.Role](credentials.Role).
				Case(models.CredentialsRoleAdmin, models.RoleAdmin).
				Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
				Default(models.RoleUser),
		},
		RefreshTokenID: &refreshTokenClaims.Jti,
	})
	if err != nil {
		span.SetData("issueToken.error", err.Error())

		return nil, NewErrLoginService(fmt.Errorf("issue accessToken: %w", err))
	}

	return &models.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
