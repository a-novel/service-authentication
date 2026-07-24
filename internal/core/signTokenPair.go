package core

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt/v2"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// tokenPairSigner is the json-keys ClaimsSign surface that signTokenPair needs.
// Service-level interfaces (e.g. TokenCreateServiceSignClaims) already match this shape,
// so callers pass their existing dependency directly.
type tokenPairSigner interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}

// signTokenPair issues a fresh refresh+access token pair for the given credentials.
//
// The two tokens are bound: the access token's RefreshTokenID claim equals the refresh
// token's JTI, so revoking the refresh token effectively revokes every access token
// derived from it. See AccessTokenClaims for the binding semantics.
//
// signTokenPair returns plain errors for the caller to report on its own span. Every
// failure path here is infrastructure failure — the json-keys RPC being down, a marshal
// error — so the helper produces no sentinels.
func signTokenPair(
	ctx context.Context, signer tokenPairSigner, credentials *dao.Credentials,
) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.signTokenPair")
	defer span.End()
	refreshTokenPayload, err := grpcf.MarshalJSONAsAny(RefreshTokenClaimsForm{
		UserID: credentials.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal refresh claims: %w", err)
	}

	refreshToken, err := signer.ClaimsSign(ctx, &servicejsonkeys.ClaimsSignRequest{
		Usage:   servicejsonkeys.KeyUsageAuthRefresh,
		Payload: refreshTokenPayload,
	})
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	// The refresh token comes straight from the trusted internal signer above, so its
	// issuer is trusted by construction and DecodeUnverified is safe on this input. Only
	// the JTI is needed, to embed in the access token below.
	refreshTokenRecipient := jwt.NewRecipient(jwt.RecipientConfig{})

	var refreshTokenClaims RefreshTokenClaims

	err = refreshTokenRecipient.DecodeUnverified(refreshToken.GetToken(), &refreshTokenClaims)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	accessTokenPayload, err := grpcf.MarshalJSONAsAny(AccessTokenClaims{
		UserID:         &credentials.ID,
		Roles:          []string{credentials.Role},
		RefreshTokenID: refreshTokenClaims.Jti,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal access claims: %w", err)
	}

	accessToken, err := signer.ClaimsSign(ctx, &servicejsonkeys.ClaimsSignRequest{
		Usage:   servicejsonkeys.KeyUsageAuth,
		Payload: accessTokenPayload,
	})
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &Token{
		AccessToken:  accessToken.GetToken(),
		RefreshToken: refreshToken.GetToken(),
	}, nil
}
