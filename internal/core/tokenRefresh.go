package core

import (
	"context"
	"errors"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2/jwp"
	"github.com/a-novel-kit/jwt/v2/jws"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

var (
	// ErrTokenRefreshInvalidAccessToken is returned by [TokenRefresh.Exec] when the
	// access token fails signature or claim verification, joined onto the underlying
	// verifier error. Expiry is ignored on the access token, since refresh exists to
	// renew an expired one.
	ErrTokenRefreshInvalidAccessToken = errors.New("invalid access token")
	// ErrTokenRefreshInvalidRefreshToken is returned by [TokenRefresh.Exec] when the
	// refresh token fails signature or claim verification, expiry included, joined
	// onto the underlying verifier error.
	ErrTokenRefreshInvalidRefreshToken = errors.New("invalid refresh token")
	// ErrTokenRefreshMismatchClaims is returned by [TokenRefresh.Exec] when the access
	// and refresh tokens are individually valid but encode different user IDs, which
	// typically means a stolen refresh token paired with someone else's access token.
	ErrTokenRefreshMismatchClaims = errors.New("refresh token claims don't match access token")
	// ErrTokenRefreshMismatchSource is returned by [TokenRefresh.Exec] when the access
	// token's RefreshTokenID claim does not match the refresh token's JTI. An access
	// token may only be renewed by the refresh token that minted it.
	ErrTokenRefreshMismatchSource = errors.New("refresh token not issued from access token")
)

// TokenRefreshDao reloads the current credentials of the user being refreshed.
type TokenRefreshDao interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

// TokenRefreshServiceSignClaims signs the new access token.
type TokenRefreshServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}

// TokenRefreshServiceVerifyClaims verifies the incoming access token and decodes its
// claims. Satisfied by servicejsonkeys.ClaimsVerifier, whose method name is the published
// client's, not ours to rename to Exec.
//
// nosemgrep: agora-dep-interface-method-must-be-exec
type TokenRefreshServiceVerifyClaims interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*AccessTokenClaims, error)
}

// TokenRefreshServiceVerifyRefreshClaims verifies the incoming refresh token and decodes
// its claims. Same published-client method name as [TokenRefreshServiceVerifyClaims].
//
// nosemgrep: agora-dep-interface-method-must-be-exec
type TokenRefreshServiceVerifyRefreshClaims interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*RefreshTokenClaims, error)
}

// TokenRefreshRequest carries the access/refresh token pair to be renewed.
type TokenRefreshRequest struct {
	AccessToken  string `validate:"required,max=1024"`
	RefreshToken string `validate:"required,max=1024"`
}

// TokenRefresh renews an access token from a valid refresh token, minting a new access
// token that reflects the user's current roles while reusing the same refresh token.
type TokenRefresh struct {
	dao                        TokenRefreshDao
	serviceSignClaims          TokenRefreshServiceSignClaims
	serviceVerifyClaims        TokenRefreshServiceVerifyClaims
	serviceVerifyRefreshClaims TokenRefreshServiceVerifyRefreshClaims
}

func NewTokenRefresh(
	dao TokenRefreshDao,
	serviceSignClaims TokenRefreshServiceSignClaims,
	serviceVerifyClaims TokenRefreshServiceVerifyClaims,
	serviceVerifyRefreshClaims TokenRefreshServiceVerifyRefreshClaims,
) *TokenRefresh {
	return &TokenRefresh{
		dao:                        dao,
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

	// Expiry is ignored here: refresh exists to renew an expired access token, but its
	// signature must still hold.
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
			return nil, otel.ReportError(span, errors.Join(err, ErrTokenRefreshInvalidAccessToken))
		}

		return nil, otel.ReportError(span, err)
	}

	// The refresh token is verified in full, expiry included.
	refreshTokenClaims, err := service.serviceVerifyRefreshClaims.VerifyClaims(
		ctx,
		&servicejsonkeys.VerifyClaimsRequest{
			Usage:       servicejsonkeys.KeyUsageAuthRefresh,
			AccessToken: request.RefreshToken,
		},
	)
	if err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) || errors.Is(err, jwp.ErrInvalidClaims) {
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

	// Binding an access token to the refresh token that minted it means revoking the
	// refresh token invalidates every access token derived from it.
	if accessTokenClaims.RefreshTokenID != refreshTokenClaims.Jti {
		return nil, otel.ReportError(span, ErrTokenRefreshMismatchSource)
	}

	// Reload credentials so any role change since the original sign lands in the new token.
	credentials, err := service.dao.Exec(ctx, &dao.CredentialsSelectRequest{
		ID: lo.FromPtr(accessTokenClaims.UserID),
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	newAccessTokenClaims, err := grpcf.MarshalJSONAsAny(AccessTokenClaims{
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
