package services

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jws"

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
// signTokenPair returns plain errors; callers wrap them with otel.ReportError on their own
// span. Sentinel errors are not produced by this helper — every failure path here is
// genuine infrastructure failure (json-keys RPC down, marshal failure, etc.).
func signTokenPair(
	ctx context.Context, signer tokenPairSigner, credentials *dao.Credentials,
) (*Token, error) {
	refreshTokenPayload, err := grpcf.InterfaceToProtoAny(RefreshTokenClaimsForm{
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

	// Parse the refresh token's claims without verifying the signature: we just obtained
	// it from the trusted internal signer (json-keys) on the previous line, so the issuer
	// is trusted by construction. We only need the JTI to embed in the access token below;
	// the access token's signature is what binds the pair end-to-end. NewInsecureVerifier
	// is safe here precisely because the input did not come from an external party.
	refreshTokenRecipient := jwt.NewRecipient(jwt.RecipientConfig{
		Plugins: []jwt.RecipientPlugin{jws.NewInsecureVerifier()},
	})

	var refreshTokenClaims RefreshTokenClaims

	err = refreshTokenRecipient.Consume(ctx, refreshToken.GetToken(), &refreshTokenClaims)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token: %w", err)
	}

	accessTokenPayload, err := grpcf.InterfaceToProtoAny(AccessTokenClaims{
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
