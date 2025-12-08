package services

import (
	"context"
	"errors"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/v2/internal/dao"
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
	ClaimsSign(
		ctx context.Context, req *jkpkg.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*jkpkg.ClaimsSignResponse, error)
}
type TokenRefreshServiceVerifyClaims interface {
	VerifyClaims(ctx context.Context, req *jkpkg.VerifyClaimsRequest) (*AccessTokenClaims, error)
}
type TokenRefreshServiceVerifyRefreshClaims interface {
	VerifyClaims(ctx context.Context, req *jkpkg.VerifyClaimsRequest) (*RefreshTokenClaims, error)
}

type TokenRefreshRequest struct {
	AccessToken  string `validate:"required,max=1024"`
	RefreshToken string `validate:"required,max=1024"`
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
		&jkpkg.VerifyClaimsRequest{
			Usage:       jkpkg.KeyUsageAuth,
			AccessToken: request.AccessToken,
			Options:     &jkpkg.VerifyClaimsOptions{IgnoreExpired: true},
		},
	)
	if err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) {
			return nil, otel.ReportError(span, errors.Join(err, ErrTokenRefreshInvalidAccessToken))
		}

		return nil, otel.ReportError(span, err)
	}

	refreshTokenClaims, err := service.serviceVerifyRefreshClaims.VerifyClaims(
		ctx,
		&jkpkg.VerifyClaimsRequest{
			Usage:       jkpkg.KeyUsageAuthRefresh,
			AccessToken: request.RefreshToken,
		},
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

	newAccessTokenClaims, err := grpcf.InterfaceToProtoAny(AccessTokenClaims{
		UserID:         accessTokenClaims.UserID,
		Roles:          []string{credentials.Role},
		RefreshTokenID: refreshTokenClaims.Jti,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	newAccessToken, err := service.serviceSignClaims.ClaimsSign(
		ctx,
		&jkpkg.ClaimsSignRequest{
			Usage:   jkpkg.KeyUsageAuth,
			Payload: newAccessTokenClaims,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return otel.ReportSuccess(span, &Token{
		AccessToken:  newAccessToken.GetToken(),
		RefreshToken: request.RefreshToken, // Refresh token does not change.
	}), nil
}
