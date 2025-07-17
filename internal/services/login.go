package services

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	jkmodels "github.com/a-novel/service-json-keys/models"
	jkpkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

// LoginSource is the source used to perform the LoginService.Login action.
//
// You may build one using the NewLoginServiceSource function.
type LoginSource interface {
	SelectCredentialsByEmail(ctx context.Context, email string) (*dao.CredentialsEntity, error)
	SignClaims(ctx context.Context, usage jkmodels.KeyUsage, claims any) (string, error)
}

func NewLoginServiceSource(
	selectCredentialsByEmailDAO *dao.SelectCredentialsByEmailRepository,
	issueTokenService *jkpkg.ClaimsSigner,
) LoginSource {
	return &struct {
		*dao.SelectCredentialsByEmailRepository
		*jkpkg.ClaimsSigner
	}{
		SelectCredentialsByEmailRepository: selectCredentialsByEmailDAO,
		ClaimsSigner:                       issueTokenService,
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
	ctx, span := otel.Tracer().Start(ctx, "service.Login")
	defer span.End()

	span.SetAttributes(attribute.String("email", request.Email))

	// Retrieve credentials.
	credentials, err := service.source.SelectCredentialsByEmail(ctx, request.Email)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select credentials by email: %w", err))
	}

	span.SetAttributes(
		attribute.String("credentials.id", credentials.ID.String()),
		attribute.String("credentials.role", credentials.Role.String()),
	)

	// Validate password.
	err = lib.CompareScrypt(request.Password, credentials.Password)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("compare password: %w", err))
	}

	refreshToken, err := service.source.SignClaims(
		ctx,
		jkmodels.KeyUsageRefresh,
		models.RefreshTokenClaimsInput{
			UserID: credentials.ID,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue refresh token: %w", err))
	}

	refreshTokenRecipient := jwt.NewRecipient(jwt.RecipientConfig{
		Plugins: []jwt.RecipientPlugin{jws.NewInsecureVerifier()},
	})

	var refreshTokenClaims models.RefreshTokenClaims

	err = refreshTokenRecipient.Consume(ctx, refreshToken, &refreshTokenClaims)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("unmarshal refresh token claims: %w", err))
	}

	// Generate a new authentication token.
	accessToken, err := service.source.SignClaims(
		ctx,
		jkmodels.KeyUsageAuth,
		models.AccessTokenClaims{
			UserID: &credentials.ID,
			Roles: []models.Role{
				lo.Switch[models.CredentialsRole, models.Role](credentials.Role).
					Case(models.CredentialsRoleAdmin, models.RoleAdmin).
					Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
					Default(models.RoleUser),
			},
			RefreshTokenID: &refreshTokenClaims.Jti,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue accessToken: %w", err))
	}

	return otel.ReportSuccess(span, &models.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}), nil
}
