package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"golang.org/x/crypto/ed25519"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwk"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/models"
)

var (
	ErrRefreshRefreshToken = errors.New(
		"only access token issued from a direct login can be used to generate a refresh token",
	)
	ErrRefreshTokenWithAnonSession = errors.New("anonymous sessions cannot issue a refresh token")

	ErrIssueRefreshTokenService = errors.New("IssueRefreshTokenService.IssueRefreshToken")
)

func NewErrIssueRefreshTokenService(err error) error {
	return errors.Join(err, ErrIssueRefreshTokenService)
}

type IssueRefreshTokenRequest struct {
	Claims *models.AccessTokenClaims
}

type IssueRefreshTokenService struct {
	producer     *jwt.Producer
	claimsConfig jwt.ClaimsProducerConfig
}

func (service *IssueRefreshTokenService) IssueRefreshToken(
	ctx context.Context, request IssueRefreshTokenRequest,
) (string, error) {
	if request.Claims.RefreshTokenID != nil {
		return "", NewErrIssueRefreshTokenService(ErrRefreshRefreshToken)
	}

	if request.Claims.UserID == nil {
		return "", NewErrIssueRefreshTokenService(ErrRefreshTokenWithAnonSession)
	}

	customClaims := models.RefreshTokenClaims{
		UserID: lo.FromPtr(request.Claims.UserID),
	}

	claims, err := jwt.NewBasicClaims(customClaims, service.claimsConfig)
	if err != nil {
		return "", NewErrIssueRefreshTokenService(fmt.Errorf("create claims: %w", err))
	}

	refreshToken, err := service.producer.Issue(ctx, claims, nil)
	if err != nil {
		return "", NewErrIssueRefreshTokenService(fmt.Errorf("issue token: %w", err))
	}

	return refreshToken, nil
}

func NewIssueRefreshTokenService(authSignSource *jwk.Source[ed25519.PrivateKey]) *IssueRefreshTokenService {
	signer := jws.NewSourcedED25519Signer(authSignSource)

	producer := jwt.NewProducer(jwt.ProducerConfig{
		Plugins: []jwt.ProducerPlugin{signer},
	})

	basicClaims := jwt.ClaimsProducerConfig{
		TargetConfig: jwt.TargetConfig{
			Issuer:   config.Tokens.Usages[models.KeyUsageRefresh].Issuer,
			Audience: config.Tokens.Usages[models.KeyUsageRefresh].Audience,
			Subject:  config.Tokens.Usages[models.KeyUsageRefresh].Subject,
		},
		TTL: config.Tokens.Usages[models.KeyUsageRefresh].TTL,
	}

	return &IssueRefreshTokenService{
		producer:     producer,
		claimsConfig: basicClaims,
	}
}
