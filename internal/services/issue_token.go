package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/ed25519"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwk"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/models"
)

var ErrIssueTokenService = errors.New("IssueTokenService.IssueToken")

func NewErrIssueTokenService(err error) error {
	return errors.Join(err, ErrIssueTokenService)
}

// IssueTokenRequest is the input used to perform the IssueTokenService.IssueToken action.
type IssueTokenRequest struct {
	// ID of the user that issued the request. Can be empty for anonymous tokens.
	UserID *uuid.UUID
	// Roles of the user that issued the request. Those are retrieved from services, and will serve to control
	// access to specific resources.
	Roles []models.Role
	// Set this value to the refresh token used to issue the new access token, if any.
	RefreshTokenID *string
}

// IssueTokenService is the service used to perform the IssueTokenService.IssueToken action.
//
// You may create one using the NewIssueTokenService function.
type IssueTokenService struct {
	producer     *jwt.Producer
	claimsConfig jwt.ClaimsProducerConfig
}

// IssueToken issues a new auth data for accessing the APIs. This data can then be passed to the AuthenticateService
// to verify the user's identity.
func (service *IssueTokenService) IssueToken(
	ctx context.Context, request IssueTokenRequest,
) (string, error) {
	customClaims := models.AccessTokenClaims{
		UserID:         request.UserID,
		Roles:          request.Roles,
		RefreshTokenID: request.RefreshTokenID,
	}

	claims, err := jwt.NewBasicClaims(customClaims, service.claimsConfig)
	if err != nil {
		return "", NewErrIssueTokenService(fmt.Errorf("create claims: %w", err))
	}

	accessToken, err := service.producer.Issue(ctx, claims, nil)
	if err != nil {
		return "", NewErrIssueTokenService(fmt.Errorf("issue token: %w", err))
	}

	return accessToken, nil
}

func NewIssueTokenService(authSignSource *jwk.Source[ed25519.PrivateKey]) *IssueTokenService {
	signer := jws.NewSourcedED25519Signer(authSignSource)

	producer := jwt.NewProducer(jwt.ProducerConfig{
		Plugins: []jwt.ProducerPlugin{signer},
	})

	basicClaims := jwt.ClaimsProducerConfig{
		TargetConfig: jwt.TargetConfig{
			Issuer:   config.Tokens.Usages[models.KeyUsageAuth].Issuer,
			Audience: config.Tokens.Usages[models.KeyUsageAuth].Audience,
			Subject:  config.Tokens.Usages[models.KeyUsageAuth].Subject,
		},
		TTL: config.Tokens.Usages[models.KeyUsageAuth].TTL,
	}

	return &IssueTokenService{
		producer:     producer,
		claimsConfig: basicClaims,
	}
}
