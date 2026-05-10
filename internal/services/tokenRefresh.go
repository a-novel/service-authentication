package services

import (
	"context"
	"errors"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/jws"
	"github.com/a-novel-kit/jwt/jwp"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

var (
	// ErrTokenRefreshInvalidAccessToken is returned by [TokenRefresh.Exec] when the
	// access token fails verification with a user-input failure: a bad signature
	// (jws.ErrInvalidSignature) or a failed claim check (jwp.ErrInvalidClaims —
	// audience, issuer, or subject mismatch; expiry is intentionally ignored on the
	// access token because refresh exists precisely to renew an expired one). The
	// sentinel is joined onto the underlying verifier error.
	ErrTokenRefreshInvalidAccessToken = errors.New("invalid access token")
	// ErrTokenRefreshInvalidRefreshToken is returned by [TokenRefresh.Exec] when
	// the refresh token fails verification with a user-input failure: a bad
	// signature (jws.ErrInvalidSignature) or a failed claim check
	// (jwp.ErrInvalidClaims — expiry, audience, issuer, or subject mismatch).
	// The sentinel is joined onto the underlying verifier error.
	ErrTokenRefreshInvalidRefreshToken = errors.New("invalid refresh token")
	// ErrTokenRefreshMismatchClaims is returned by [TokenRefresh.Exec] when the
	// access and refresh tokens are individually valid but encode different user
	// IDs. This typically indicates an attacker pairing a stolen refresh token
	// with someone else's access token.
	ErrTokenRefreshMismatchClaims = errors.New("refresh token claims don't match access token")
	// ErrTokenRefreshMismatchSource is returned by [TokenRefresh.Exec] when the
	// access token's RefreshTokenID claim does not match the refresh token's JTI.
	// This breaks the access/refresh binding — an access token can only be renewed
	// using the very refresh token that originally minted it.
	ErrTokenRefreshMismatchSource = errors.New("refresh token not issued from access token")
)

type TokenRefreshRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}
type TokenRefreshServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}
type TokenRefreshServiceVerifyClaims interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*AccessTokenClaims, error)
}
type TokenRefreshServiceVerifyRefreshClaims interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*RefreshTokenClaims, error)
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
		return nil, errors.Join(err, ErrInvalidRequest)
	}

	// Step 1: Verify the access token signature.
	// We allow expired tokens here because the whole point of refresh is to get a new access token
	// after the old one expires. However, the signature must still be valid to prevent forgery.
	accessTokenClaims, err := service.serviceVerifyClaims.VerifyClaims(
		ctx,
		&servicejsonkeys.VerifyClaimsRequest{
			Usage:       servicejsonkeys.KeyUsageAuth,
			AccessToken: request.AccessToken,
			Options:     &servicejsonkeys.VerifyClaimsOptions{IgnoreExpired: true},
		},
	)
	if err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) || errors.Is(err, jwp.ErrInvalidClaims) {
			return nil, errors.Join(err, ErrTokenRefreshInvalidAccessToken)
		}

		return nil, otel.ReportError(span, err)
	}

	// Step 2: Verify the refresh token signature and expiration.
	// Unlike access tokens, refresh tokens must not be expired.
	refreshTokenClaims, err := service.serviceVerifyRefreshClaims.VerifyClaims(
		ctx,
		&servicejsonkeys.VerifyClaimsRequest{
			Usage:       servicejsonkeys.KeyUsageAuthRefresh,
			AccessToken: request.RefreshToken,
		},
	)
	if err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) || errors.Is(err, jwp.ErrInvalidClaims) {
			return nil, errors.Join(err, ErrTokenRefreshInvalidRefreshToken)
		}

		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(
		attribute.String("accessTokenClaims.userID", lo.FromPtr(accessTokenClaims.UserID).String()),
		attribute.String("refreshTokenClaims.userID", refreshTokenClaims.UserID.String()),
	)

	// Step 3: Verify that both tokens belong to the same user.
	// This prevents an attacker from using a stolen refresh token with a different user's access token.
	if lo.FromPtr(accessTokenClaims.UserID) != refreshTokenClaims.UserID {
		return nil, ErrTokenRefreshMismatchClaims
	}

	span.SetAttributes(
		attribute.String("accessTokenClaims.refreshTokenID", accessTokenClaims.RefreshTokenID),
		attribute.String("refreshTokenClaims.jti", refreshTokenClaims.Jti),
	)

	// Step 4: Verify that the access token was issued from this specific refresh token.
	// Each access token stores the JTI of the refresh token that created it.
	// This binding ensures that:
	//   - An access token can only be refreshed using its original refresh token
	//   - Revoking a refresh token effectively revokes all access tokens derived from it
	if accessTokenClaims.RefreshTokenID != refreshTokenClaims.Jti {
		return nil, ErrTokenRefreshMismatchSource
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
		&servicejsonkeys.ClaimsSignRequest{
			Usage:   servicejsonkeys.KeyUsageAuth,
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
