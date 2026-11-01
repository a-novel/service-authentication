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
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// tokenPairSigner is the JSON Keys signing surface signTokenPair needs. The signing
// dependency of each calling operation already matches it.
type tokenPairSigner interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}

// signTokenPair issues a fresh refresh and access token pair for the given credentials, opening
// a session of their own. The access token's RefreshTokenID claim holds the refresh token's JTI,
// which binds the pair, and its SessionID the session. Every failure is an infrastructure fault,
// so the function returns no sentinel.
func signTokenPair(ctx context.Context, signer tokenPairSigner, credentials *dao.Credentials) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.signTokenPair")
	defer span.End()

	refreshTokenPayload, err := grpcf.MarshalJSONAsAny(session.RefreshClaimsForm{UserID: credentials.ID})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("marshal refresh claims: %w", err))
	}

	refreshTokenRequest := &servicejsonkeys.ClaimsSignRequest{
		Usage:   servicejsonkeys.KeyUsageAuthRefresh,
		Payload: refreshTokenPayload,
	}

	refreshToken, err := signer.ClaimsSign(ctx, refreshTokenRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue refresh token: %w", err))
	}

	// The refresh token comes straight from the trusted internal signer above, so its
	// issuer is trusted by construction and DecodeUnverified is safe on this input. Only
	// the JTI and the session are needed, to embed in the access token below.
	refreshTokenRecipient := jwt.NewRecipient(jwt.RecipientConfig{})

	var refreshTokenClaims session.RefreshClaims

	err = refreshTokenRecipient.DecodeUnverified(refreshToken.GetToken(), &refreshTokenClaims)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("parse refresh token: %w", err))
	}

	accessTokenClaims := session.Claims{
		UserID:         &credentials.ID,
		Roles:          []string{credentials.Role},
		RefreshTokenID: refreshTokenClaims.Jti,
		SessionID:      refreshTokenClaims.Session(),
	}

	accessTokenPayload, err := grpcf.MarshalJSONAsAny(accessTokenClaims)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("marshal access claims: %w", err))
	}

	accessTokenRequest := &servicejsonkeys.ClaimsSignRequest{
		Usage:   servicejsonkeys.KeyUsageAuth,
		Payload: accessTokenPayload,
	}

	accessToken, err := signer.ClaimsSign(ctx, accessTokenRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue access token: %w", err))
	}

	return &Token{
		AccessToken:  accessToken.GetToken(),
		RefreshToken: refreshToken.GetToken(),
	}, nil
}
