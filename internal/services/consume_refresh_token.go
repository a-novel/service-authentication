package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	jkModels "github.com/a-novel/service-json-keys/models"
	jkPkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

var (
	ErrMismatchRefreshClaims                = errors.New("refresh token and access token do not match")
	ErrTokenIssuedWithDifferentRefreshToken = errors.New("access token was not issued with the provided refresh token")
)

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
	ctx, span := otel.Tracer().Start(ctx, "service.ConsumeRefreshToken")
	defer span.End()

	if request.AccessToken == "" {
		return "", otel.ReportError(span, fmt.Errorf("%w: access token is empty", models.ErrUnauthorized))
	}

	if request.RefreshToken == "" {
		return "", otel.ReportError(span, fmt.Errorf("%w: refresh token is empty", models.ErrUnauthorized))
	}

	accessTokenClaims, err := service.source.VerifyClaims(
		ctx,
		jkModels.KeyUsageAuth,
		request.AccessToken,
		&jkPkg.VerifyClaimsOptions{IgnoreExpired: true},
	)
	if err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) {
			return "", otel.ReportError(span, fmt.Errorf("consume access token: %w", models.ErrUnauthorized))
		}

		return "", otel.ReportError(span, fmt.Errorf("verify access token claims: %w", err))
	}

	refreshTokenClaims, err := service.source.VerifyRefreshTokenClaims(
		ctx,
		jkModels.KeyUsageRefresh,
		request.RefreshToken,
		nil,
	)
	if err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) {
			return "", otel.ReportError(span, fmt.Errorf("consume refresh token: %w", models.ErrUnauthorized))
		}

		return "", otel.ReportError(span, fmt.Errorf("consume refresh token: %w", err))
	}

	span.SetAttributes(
		attribute.String("accessTokenClaims.userID", lo.FromPtr(accessTokenClaims.UserID).String()),
		attribute.String("refreshTokenClaims.userID", refreshTokenClaims.UserID.String()),
	)

	if lo.FromPtr(accessTokenClaims.UserID) != refreshTokenClaims.UserID {
		return "", otel.ReportError(span, fmt.Errorf(
			"%w (accessToken.userID: %s, refreshToken.userID: %s)",
			ErrMismatchRefreshClaims,
			lo.FromPtr(accessTokenClaims.UserID),
			refreshTokenClaims.UserID,
		))
	}

	span.SetAttributes(
		attribute.String("accessTokenClaims.refreshTokenID", lo.FromPtr(accessTokenClaims.RefreshTokenID)),
		attribute.String("refreshTokenClaims.jti", refreshTokenClaims.Jti),
	)

	if lo.FromPtr(accessTokenClaims.RefreshTokenID) != refreshTokenClaims.Jti {
		return "", otel.ReportError(span, ErrTokenIssuedWithDifferentRefreshToken)
	}

	if lo.FromPtr(accessTokenClaims.UserID) != refreshTokenClaims.UserID {
		return "", otel.ReportError(span, fmt.Errorf(
			"%w (accessToken.userID: %s, refreshToken.userID: %s)",
			ErrMismatchRefreshClaims,
			lo.FromPtr(accessTokenClaims.UserID),
			refreshTokenClaims.UserID,
		))
	}

	// Retrieve updated credentials.
	credentials, err := service.source.SelectCredentials(ctx, lo.FromPtr(accessTokenClaims.UserID))
	if err != nil {
		return "", otel.ReportError(span, fmt.Errorf("select credentials: %w", err))
	}

	newAccessToken, err := service.source.SignClaims(ctx, jkModels.KeyUsageAuth, &models.AccessTokenClaims{
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
		return "", otel.ReportError(span, fmt.Errorf("issue accessToken: %w", err))
	}

	return otel.ReportSuccess(span, newAccessToken), nil
}
