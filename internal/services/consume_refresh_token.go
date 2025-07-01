package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"golang.org/x/crypto/ed25519"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwk"
	"github.com/a-novel-kit/jwt/jwp"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

var (
	ErrMismatchRefreshClaims                = errors.New("refresh token and access token do not match")
	ErrTokenIssuedWithDifferentRefreshToken = errors.New("access token was not issued with the provided refresh token")

	ErrConsumeRefreshTokenService = errors.New("ConsumeRefreshTokenService.ConsumeRefreshToken")
)

func NewErrConsumeRefreshTokenService(err error) error {
	return errors.Join(err, ErrConsumeRefreshTokenService)
}

type ConsumeRefreshTokenSource interface {
	SelectCredentials(ctx context.Context, id uuid.UUID) (*dao.CredentialsEntity, error)
	IssueToken(ctx context.Context, request IssueTokenRequest) (string, error)
}

func NewConsumeRefreshTokenServiceSource(
	selectCredentialsDAO *dao.SelectCredentialsRepository,
	issueTokenService *IssueTokenService,
) ConsumeRefreshTokenSource {
	return &struct {
		*dao.SelectCredentialsRepository
		*IssueTokenService
	}{
		SelectCredentialsRepository: selectCredentialsDAO,
		IssueTokenService:           issueTokenService,
	}
}

type ConsumeRefreshTokenRequest struct {
	// The last valid access token must be provided as an extra security measure. It may be expired, but must still
	// be signed by an active key. Information in the access token is checked against the refresh token claims, and
	// used to populate the new access token claims.
	AccessToken  string
	RefreshToken string
}

type ConsumeRefreshTokenService struct {
	source                ConsumeRefreshTokenSource
	accessTokenRecipient  *jwt.Recipient
	refreshTokenRecipient *jwt.Recipient
}

func NewConsumeRefreshTokenService(
	source ConsumeRefreshTokenSource,
	accessTokenKeysSource *jwk.Source[ed25519.PublicKey],
	refreshTokenKeysSource *jwk.Source[ed25519.PublicKey],
) *ConsumeRefreshTokenService {
	accessTokenVerifier := jws.NewSourcedED25519Verifier(accessTokenKeysSource)
	refreshTokenVerifier := jws.NewSourcedED25519Verifier(refreshTokenKeysSource)

	accessTokenDeserializer := jwp.NewClaimsChecker(&jwp.ClaimsCheckerConfig{
		Checks: []jwp.ClaimsCheck{
			jwp.NewClaimsCheckTarget(jwt.TargetConfig{
				Issuer:   config.Tokens.Usages[models.KeyUsageAuth].Issuer,
				Audience: config.Tokens.Usages[models.KeyUsageAuth].Audience,
				Subject:  config.Tokens.Usages[models.KeyUsageAuth].Subject,
			}),
			// Ignore timestamps checks. We just need to ensure this service issued the token, not that it's
			// still valid.
		},
	})
	refreshTokenDeserializer := jwp.NewClaimsChecker(&jwp.ClaimsCheckerConfig{
		Checks: []jwp.ClaimsCheck{
			jwp.NewClaimsCheckTarget(jwt.TargetConfig{
				Issuer:   config.Tokens.Usages[models.KeyUsageRefresh].Issuer,
				Audience: config.Tokens.Usages[models.KeyUsageRefresh].Audience,
				Subject:  config.Tokens.Usages[models.KeyUsageRefresh].Subject,
			}),
			jwp.NewClaimsCheckTimestamp(config.Tokens.Usages[models.KeyUsageRefresh].Leeway, true),
		},
	})

	return &ConsumeRefreshTokenService{
		source: source,
		accessTokenRecipient: jwt.NewRecipient(jwt.RecipientConfig{
			Plugins:      []jwt.RecipientPlugin{accessTokenVerifier},
			Deserializer: accessTokenDeserializer.Unmarshal,
		}),
		refreshTokenRecipient: jwt.NewRecipient(jwt.RecipientConfig{
			Plugins:      []jwt.RecipientPlugin{refreshTokenVerifier},
			Deserializer: refreshTokenDeserializer.Unmarshal,
		}),
	}
}

func (service *ConsumeRefreshTokenService) ConsumeRefreshToken(
	ctx context.Context, request ConsumeRefreshTokenRequest,
) (string, error) {
	span := sentry.StartSpan(ctx, "ConsumeRefreshTokenService.ConsumeRefreshToken")
	defer span.Finish()

	if request.AccessToken == "" {
		span.SetData("error", "access token is empty")

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf("%w: access token is empty", models.ErrUnauthorized))
	}

	if request.RefreshToken == "" {
		span.SetData("error", "refresh token is empty")

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf("%w: refresh token is empty", models.ErrUnauthorized))
	}

	var accessTokenClaims models.AccessTokenClaims

	err := service.accessTokenRecipient.Consume(span.Context(), request.AccessToken, &accessTokenClaims)
	if err != nil {
		span.SetData("accessToken.consume.error", err.Error())

		if errors.Is(err, jws.ErrInvalidSignature) {
			return "", NewErrConsumeRefreshTokenService(fmt.Errorf("consume access token: %w", models.ErrUnauthorized))
		}

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf("consume access token: %w", err))
	}

	var refreshTokenClaims models.RefreshTokenClaims

	err = service.refreshTokenRecipient.Consume(span.Context(), request.RefreshToken, &refreshTokenClaims)
	if err != nil {
		span.SetData("refreshToken.consume.error", err.Error())

		if errors.Is(err, jws.ErrInvalidSignature) {
			return "", NewErrConsumeRefreshTokenService(fmt.Errorf("consume refresh token: %w", models.ErrUnauthorized))
		}

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf("consume refresh token: %w", err))
	}

	span.SetData("accessTokenClaims.userID", lo.FromPtr(accessTokenClaims.UserID))
	span.SetData("refreshTokenClaims.userID", refreshTokenClaims.UserID)

	if lo.FromPtr(accessTokenClaims.UserID) != refreshTokenClaims.UserID {
		span.SetData("error", "access token userID does not match refresh token userID")

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf(
			"%w (accessToken.userID: %s, refreshToken.userID: %s)",
			ErrMismatchRefreshClaims,
			lo.FromPtr(accessTokenClaims.UserID),
			refreshTokenClaims.UserID,
		))
	}

	span.SetData("accessTokenClaims.refreshTokenID", lo.FromPtr(accessTokenClaims.RefreshTokenID))
	span.SetData("refreshTokenClaims.jti", refreshTokenClaims.Jti)

	if lo.FromPtr(accessTokenClaims.RefreshTokenID) != refreshTokenClaims.Jti {
		span.SetData("error", "access token was not issued with the provided refresh token")

		return "", NewErrConsumeRefreshTokenService(ErrTokenIssuedWithDifferentRefreshToken)
	}

	// Retrieve updated credentials.
	if accessTokenClaims.UserID != nil {
		credentials, err := service.source.SelectCredentials(ctx, lo.FromPtr(accessTokenClaims.UserID))
		if err != nil {
			span.SetData("selectCredentials.error", err.Error())

			return "", NewErrConsumeRefreshTokenService(fmt.Errorf("select credentials: %w", err))
		}

		accessTokenClaims.Roles = []models.Role{
			lo.Switch[models.CredentialsRole, models.Role](credentials.Role).
				Case(models.CredentialsRoleAdmin, models.RoleAdmin).
				Case(models.CredentialsRoleSuperAdmin, models.RoleSuperAdmin).
				Default(models.RoleUser),
		}
	}

	accessToken, err := service.source.IssueToken(span.Context(), IssueTokenRequest{
		UserID:         accessTokenClaims.UserID,
		Roles:          accessTokenClaims.Roles,
		RefreshTokenID: &refreshTokenClaims.Jti,
	})
	if err != nil {
		span.SetData("issueToken.error", err.Error())

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf("issue accessToken: %w", err))
	}

	return accessToken, nil
}
