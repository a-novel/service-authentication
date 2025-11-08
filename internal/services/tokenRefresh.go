package services

import (
	"context"
	"errors"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	jkmodels "github.com/a-novel/service-json-keys/models"
	jkpkg "github.com/a-novel/service-json-keys/pkg"

	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/dao"
)

var (
	ErrTokenRefreshInvalidAccessToken  = errors.New("invalid access token")
	ErrTokenRefreshInvalidRefreshToken = errors.New("invalid refresh token")
	ErrTokenRefreshMismatchClaims      = errors.New("refresh token claims don't match access token")
	ErrTokenRefreshMismatchSource      = errors.New("refresh token not issued from access token")
)

type TokenRefreshRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}
type TokenRefreshServiceSignClaims interface {
	SignClaims(ctx context.Context, usage jkmodels.KeyUsage, claims any) (string, error)
}
type TokenRefreshServiceVerifyClaims interface {
	VerifyClaims(
		ctx context.Context, usage jkmodels.KeyUsage, accessToken string, options *jkpkg.VerifyClaimsOptions,
	) (*AccessTokenClaims, error)
}
type TokenRefreshServiceVerifyRefreshClaims interface {
	VerifyClaims(
		ctx context.Context, usage jkmodels.KeyUsage, refreshToken string, options *jkpkg.VerifyClaimsOptions,
	) (*RefreshTokenClaims, error)
}

type TokenRefreshRequest struct {
	AccessToken  string `validate:"required,max=1024,base64rawurl"`
	RefreshToken string `validate:"required,max=1024,base64rawurl"`
}

type TokenRefresh struct {
	repository                 TokenRefreshRepository
	serviceSignClaims          TokenRefreshServiceSignClaims
	serviceVerifyClaims        TokenRefreshServiceVerifyClaims
	serviceVerifyRefreshClaims TokenRefreshServiceVerifyRefreshClaims
}

func NewTokenRefresh(
	repository TokenRefreshRepository,
	serviceSignClaims TokenRefreshServiceSignClaims,
	serviceVerifyClaims TokenRefreshServiceVerifyClaims,
	serviceVerifyRefreshClaims TokenRefreshServiceVerifyRefreshClaims,
) *TokenRefresh {
	return &TokenRefresh{
		repository:                 repository,
		serviceSignClaims:          serviceSignClaims,
		serviceVerifyClaims:        serviceVerifyClaims,
		serviceVerifyRefreshClaims: serviceVerifyRefreshClaims,
	}
}

func (service *TokenRefresh) Exec(
	ctx context.Context, request *TokenRefreshRequest,
) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.TokenRefresh")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	accessTokenClaims, err := service.serviceVerifyClaims.VerifyClaims(
		ctx,
		jkmodels.KeyUsageAuth,
		request.AccessToken,
		&jkpkg.VerifyClaimsOptions{IgnoreExpired: true},
	)
	if err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) {
			return nil, otel.ReportError(span, errors.Join(err, ErrTokenRefreshInvalidAccessToken))
		}

		return nil, otel.ReportError(span, err)
	}

	refreshTokenClaims, err := service.serviceVerifyRefreshClaims.VerifyClaims(
		ctx,
		jkmodels.KeyUsageRefresh,
		request.RefreshToken,
		nil,
	)
	if err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) {
			return nil, otel.ReportError(span, errors.Join(err, ErrTokenRefreshInvalidRefreshToken))
		}

		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(
		attribute.String("accessTokenClaims.userID", lo.FromPtr(accessTokenClaims.UserID).String()),
		attribute.String("refreshTokenClaims.userID", refreshTokenClaims.UserID.String()),
	)

	if lo.FromPtr(accessTokenClaims.UserID) != refreshTokenClaims.UserID {
		return nil, otel.ReportError(span, ErrTokenRefreshMismatchClaims)
	}

	span.SetAttributes(
		attribute.String("accessTokenClaims.refreshTokenID", accessTokenClaims.RefreshTokenID),
		attribute.String("refreshTokenClaims.jti", refreshTokenClaims.Jti),
	)

	if accessTokenClaims.RefreshTokenID != refreshTokenClaims.Jti {
		return nil, otel.ReportError(span, ErrTokenRefreshMismatchSource)
	}

	// Retrieve updated credentials.
	credentials, err := service.repository.Exec(ctx, &dao.CredentialsSelectRequest{
		ID: lo.FromPtr(accessTokenClaims.UserID),
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	newAccessToken, err := service.serviceSignClaims.SignClaims(ctx, jkmodels.KeyUsageAuth, &AccessTokenClaims{
		UserID:         accessTokenClaims.UserID,
		Roles:          []string{credentials.Role},
		RefreshTokenID: refreshTokenClaims.Jti,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, &Token{
		AccessToken:  newAccessToken,
		RefreshToken: request.RefreshToken, // Refresh token does not change.
	}), nil
}
