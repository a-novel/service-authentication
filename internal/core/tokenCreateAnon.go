package core

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

// anonTokenClaims is the fixed sign request for anonymous access tokens. The payload
// never varies — no user ID, only the anonymous role — so it is marshaled once at
// package load rather than per request.
var anonTokenClaims = &servicejsonkeys.ClaimsSignRequest{
	Usage: servicejsonkeys.KeyUsageAuth,
	Payload: lo.Must(grpcf.MarshalJSONAsAny(AccessTokenClaims{
		Roles: []string{config.RoleAnon},
	})),
}

// TokenCreateAnonSignClaimsService is the json-keys signing surface TokenCreateAnon needs.
type TokenCreateAnonSignClaimsService interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}

// TokenCreateAnon issues an anonymous access token, carrying no user ID and only the
// anonymous role. See AccessTokenClaims for what such tokens grant.
type TokenCreateAnon struct {
	signClaimsService TokenCreateAnonSignClaimsService
}

func NewTokenCreateAnon(signClaimsService TokenCreateAnonSignClaimsService) *TokenCreateAnon {
	return &TokenCreateAnon{
		signClaimsService: signClaimsService,
	}
}

func (service *TokenCreateAnon) Exec(ctx context.Context) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.TokenCreateAnon")
	defer span.End()

	accessToken, err := service.signClaimsService.ClaimsSign(ctx, anonTokenClaims)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue accessToken: %w", err))
	}

	return otel.ReportSuccess(span, &Token{
		AccessToken: accessToken.GetToken(),
	}), nil
}
