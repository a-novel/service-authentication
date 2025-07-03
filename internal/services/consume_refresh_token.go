package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/samber/lo"

	jkModels "github.com/a-novel/service-json-keys/models"
	jkPkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/jwt/jws"

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
	SignClaims(ctx context.Context, usage jkModels.KeyUsage, claims any) (string, error)
	VerifyClaims(
		ctx context.Context, usage jkModels.KeyUsage, accessToken string, options *jkPkg.VerifyClaimsOptions,
	) (*models.AccessTokenClaims, error)
	VerifyRefreshTokenClaims(
		ctx context.Context, usage jkModels.KeyUsage, accessToken string, options *jkPkg.VerifyClaimsOptions,
	) (*models.RefreshTokenClaims, error)
}

func NewConsumeRefreshTokenServiceSource(
	selectCredentialsDAO *dao.SelectCredentialsRepository,
	issueTokenService *jkPkg.ClaimsSigner,
	accessTokenService *jkPkg.ClaimsVerifier[models.AccessTokenClaims],
	refreshTokenService *jkPkg.ClaimsVerifier[models.RefreshTokenClaims],
) ConsumeRefreshTokenSource {
	return &struct {
		*dao.SelectCredentialsRepository
		*jkPkg.ClaimsSigner
		*jkPkg.ClaimsVerifier[models.AccessTokenClaims]
		*RefreshTokenClaimsVerifier
	}{
		SelectCredentialsRepository: selectCredentialsDAO,
		ClaimsSigner:                issueTokenService,
		ClaimsVerifier:              accessTokenService,
		RefreshTokenClaimsVerifier: &RefreshTokenClaimsVerifier{
			verifier: refreshTokenService,
		},
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
	source ConsumeRefreshTokenSource
}

func NewConsumeRefreshTokenService(source ConsumeRefreshTokenSource) *ConsumeRefreshTokenService {
	return &ConsumeRefreshTokenService{
		source: source,
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

	accessTokenClaims, err := service.source.VerifyClaims(
		span.Context(),
		jkModels.KeyUsageAuth,
		request.AccessToken,
		&jkPkg.VerifyClaimsOptions{IgnoreExpired: true},
	)
	if err != nil {
		span.SetData("accessToken.verify.error", err.Error())
		sentry.CaptureException(err)

		if errors.Is(err, jws.ErrInvalidSignature) {
			return "", NewErrConsumeRefreshTokenService(fmt.Errorf("consume access token: %w", models.ErrUnauthorized))
		}

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf("verify access token claims: %w", err))
	}

	refreshTokenClaims, err := service.source.VerifyRefreshTokenClaims(
		span.Context(),
		jkModels.KeyUsageRefresh,
		request.RefreshToken,
		nil,
	)
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

	if lo.FromPtr(accessTokenClaims.UserID) != refreshTokenClaims.UserID {
		span.SetData("error", "access token userID does not match refresh token userID")

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf(
			"%w (accessToken.userID: %s, refreshToken.userID: %s)",
			ErrMismatchRefreshClaims,
			lo.FromPtr(accessTokenClaims.UserID),
			refreshTokenClaims.UserID,
		))
	}

	// Retrieve updated credentials.
	credentials, err := service.source.SelectCredentials(ctx, lo.FromPtr(accessTokenClaims.UserID))
	if err != nil {
		span.SetData("selectCredentials.error", err.Error())

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf("select credentials: %w", err))
	}

	newAccessToken, err := service.source.SignClaims(span.Context(), jkModels.KeyUsageAuth, &models.AccessTokenClaims{
		UserID: accessTokenClaims.UserID,
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

		return "", NewErrConsumeRefreshTokenService(fmt.Errorf("issue accessToken: %w", err))
	}

	return newAccessToken, nil
}
