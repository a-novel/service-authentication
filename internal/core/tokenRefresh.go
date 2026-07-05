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
// claims. It is distinct from TokenRefreshServiceVerifyRefreshClaims only in the claim
// type it returns.
type TokenRefreshServiceVerifyClaims interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*AccessTokenClaims, error)
}

// TokenRefreshServiceVerifyRefreshClaims verifies the incoming refresh token and decodes
// its claims.
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

	// Verify the access token's signature. Expiry is ignored on purpose: renewing an
	// expired access token is exactly what refresh is for, but the signature must hold.
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

	// Verify the refresh token. Unlike the access token, it must not be expired.
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

	// Both tokens must name the same user; see ErrTokenRefreshMismatchClaims.
	if lo.FromPtr(accessTokenClaims.UserID) != refreshTokenClaims.UserID {
		return nil, otel.ReportError(span, ErrTokenRefreshMismatchClaims)
	}

	span.SetAttributes(
		attribute.String("accessTokenClaims.refreshTokenID", accessTokenClaims.RefreshTokenID),
		attribute.String("refreshTokenClaims.jti", refreshTokenClaims.Jti),
	)

	// The access token must have been minted by this very refresh token — so revoking a
	// refresh token also invalidates every access token minted from it. See
	// ErrTokenRefreshMismatchSource for the binding this enforces.
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
