package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/samber/lo"
	"golang.org/x/crypto/ed25519"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwa"
	"github.com/a-novel-kit/jwt/jwk"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/models"
)

var (
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

func (service *IssueRefreshTokenService) IssueRefreshToken(
	ctx context.Context, request IssueRefreshTokenRequest,
) (string, *jwa.Claims, error) {
	span := sentry.StartSpan(ctx, "IssueRefreshTokenService.IssueRefreshToken")
	defer span.Finish()

	span.SetData("userID", lo.FromPtr(request.Claims.UserID))
	span.SetData("refreshTokenID", lo.FromPtr(request.Claims.RefreshTokenID))
	span.SetData("roles", request.Claims.Roles)

	if request.Claims.UserID == nil {
		span.SetData("error", "user ID is not set in the claims")

		return "", nil, NewErrIssueRefreshTokenService(ErrRefreshTokenWithAnonSession)
	}

	customClaims := map[string]any{
		"userID": lo.FromPtr(request.Claims.UserID),
	}

	claims, err := jwt.NewBasicClaims(customClaims, service.claimsConfig)
	if err != nil {
		span.SetData("claims.create.error", err.Error())

		return "", nil, NewErrIssueRefreshTokenService(fmt.Errorf("create claims: %w", err))
	}

	refreshToken, err := service.producer.Issue(span.Context(), claims, nil)
	if err != nil {
		span.SetData("jwt.issue.error", err.Error())

		return "", nil, NewErrIssueRefreshTokenService(fmt.Errorf("issue token: %w", err))
	}

	return refreshToken, claims, nil
}
