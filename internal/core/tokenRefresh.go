package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

var (
	// ErrTokenRefreshInvalidAccessToken is returned by [TokenRefresh.Exec] when the
	// verifier rejects the access token with [jwt.ErrTokenRejected], joined onto that
	// verifier error. Expiry is ignored on the access token, since refresh exists to
	// renew an expired one.
	ErrTokenRefreshInvalidAccessToken = errors.New("invalid access token")
	// ErrTokenRefreshInvalidRefreshToken is returned by [TokenRefresh.Exec] when the
	// verifier rejects the refresh token with [jwt.ErrTokenRejected], expiry included,
	// joined onto that verifier error. A revoked refresh token returns it too.
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

// TokenRefreshDaoRefreshTokenRevocationExist reports whether the refresh token was revoked.
type TokenRefreshDaoRefreshTokenRevocationExist interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationExistRequest) (bool, error)
}

// TokenRefreshServiceSignClaims signs the new access token.
type TokenRefreshServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}

// TokenRefreshServiceVerifyClaims verifies the incoming access token and decodes its
// claims. servicejsonkeys.ClaimsVerifier satisfies it, so the method keeps that published
// client's name instead of Exec.
//
// nosemgrep: a-novel-dep-interface-method-must-be-exec
type TokenRefreshServiceVerifyClaims interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*session.Claims, error)
}

// TokenRefreshServiceVerifyRefreshClaims verifies the incoming refresh token and decodes
// its claims. Same published-client method name as [TokenRefreshServiceVerifyClaims].
//
// nosemgrep: a-novel-dep-interface-method-must-be-exec
type TokenRefreshServiceVerifyRefreshClaims interface {
	VerifyClaims(ctx context.Context, req *servicejsonkeys.VerifyClaimsRequest) (*session.RefreshClaims, error)
}

// TokenRefreshRequest carries the access/refresh token pair to be renewed.
type TokenRefreshRequest struct {
	AccessToken  string `validate:"required,max=1024"`
	RefreshToken string `validate:"required,max=1024"`
}

// TokenRefresh renews an access token from a valid refresh token that was not revoked, minting a
// new access token that reflects the user's current roles while reusing the same refresh token.
type TokenRefresh struct {
	dao                            TokenRefreshDao
	daoRefreshTokenRevocationExist TokenRefreshDaoRefreshTokenRevocationExist
	serviceSignClaims              TokenRefreshServiceSignClaims
	serviceVerifyClaims            TokenRefreshServiceVerifyClaims
	serviceVerifyRefreshClaims     TokenRefreshServiceVerifyRefreshClaims
}

func NewTokenRefresh(
	dao TokenRefreshDao,
	daoRefreshTokenRevocationExist TokenRefreshDaoRefreshTokenRevocationExist,
	serviceSignClaims TokenRefreshServiceSignClaims,
	serviceVerifyClaims TokenRefreshServiceVerifyClaims,
	serviceVerifyRefreshClaims TokenRefreshServiceVerifyRefreshClaims,
) *TokenRefresh {
	return &TokenRefresh{
		dao:                            dao,
		daoRefreshTokenRevocationExist: daoRefreshTokenRevocationExist,
		serviceSignClaims:              serviceSignClaims,
		serviceVerifyClaims:            serviceVerifyClaims,
		serviceVerifyRefreshClaims:     serviceVerifyRefreshClaims,
	}
}

func (service *TokenRefresh) Exec(
	ctx context.Context, request *TokenRefreshRequest,
) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.TokenRefresh")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Expiry is ignored here: refresh exists to renew an expired access token, but its
	// signature must still hold.
	accessTokenRequest := &servicejsonkeys.VerifyClaimsRequest{
		Usage:       servicejsonkeys.KeyUsageAuth,
		AccessToken: request.AccessToken,
		Options:     &servicejsonkeys.VerifyClaimsOptions{IgnoreExpired: true},
	}

	accessTokenClaims, err := service.serviceVerifyClaims.VerifyClaims(ctx, accessTokenRequest)
	if err != nil {
		return nil, otel.ReportError(span, joinInvalidToken(err, ErrTokenRefreshInvalidAccessToken))
	}

	// The refresh token is verified in full, expiry included.
	refreshTokenRequest := &servicejsonkeys.VerifyClaimsRequest{
		Usage:       servicejsonkeys.KeyUsageAuthRefresh,
		AccessToken: request.RefreshToken,
	}

	refreshTokenClaims, err := service.serviceVerifyRefreshClaims.VerifyClaims(ctx, refreshTokenRequest)
	if err != nil {
		return nil, otel.ReportError(span, joinInvalidToken(err, ErrTokenRefreshInvalidRefreshToken))
	}

	span.SetAttributes(
		attribute.String("access_token.user_id", lo.FromPtr(accessTokenClaims.UserID).String()),
		attribute.String("refresh_token.user_id", refreshTokenClaims.UserID.String()),
	)

	if lo.FromPtr(accessTokenClaims.UserID) != refreshTokenClaims.UserID {
		return nil, otel.ReportError(span, ErrTokenRefreshMismatchClaims)
	}

	span.SetAttributes(
		attribute.String("access_token.refresh_token_id", accessTokenClaims.RefreshTokenID),
		attribute.String("refresh_token.id", refreshTokenClaims.Jti),
	)

	// Only the refresh token that minted an access token renews it.
	if accessTokenClaims.RefreshTokenID != refreshTokenClaims.Jti {
		return nil, otel.ReportError(span, ErrTokenRefreshMismatchSource)
	}

	revocationRequest := &dao.RefreshTokenRevocationExistRequest{JTI: refreshTokenClaims.Jti}

	revoked, err := service.daoRefreshTokenRevocationExist.Exec(ctx, revocationRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check refresh token revocation: %w", err))
	}

	if revoked {
		return nil, otel.ReportError(span, fmt.Errorf("refresh token revoked: %w", ErrTokenRefreshInvalidRefreshToken))
	}

	// Reload credentials so any role change since the original sign lands in the new token.
	credentials, err := service.dao.Exec(ctx, &dao.CredentialsSelectRequest{ID: lo.FromPtr(accessTokenClaims.UserID)})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	newAccessTokenClaims := session.Claims{
		UserID:         accessTokenClaims.UserID,
		Roles:          []string{credentials.Role},
		RefreshTokenID: refreshTokenClaims.Jti,
	}

	newAccessTokenPayload, err := grpcf.MarshalJSONAsAny(newAccessTokenClaims)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	signRequest := &servicejsonkeys.ClaimsSignRequest{
		Usage:   servicejsonkeys.KeyUsageAuth,
		Payload: newAccessTokenPayload,
	}

	newAccessToken, err := service.serviceSignClaims.ClaimsSign(ctx, signRequest)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return &Token{
		AccessToken:  newAccessToken.GetToken(),
		RefreshToken: request.RefreshToken, // Refresh token does not change.
	}, nil
}

// joinInvalidToken joins sentinel onto err when the verifier rejected the token itself. Any other
// failure, such as an unreachable key source, stays as it is.
func joinInvalidToken(err, sentinel error) error {
	if errors.Is(err, jwt.ErrTokenRejected) {
		return errors.Join(err, sentinel)
	}

	return err
}
