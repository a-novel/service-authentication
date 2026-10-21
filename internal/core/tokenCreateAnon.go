package core

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// anonTokenClaims is the sign request of every anonymous access token. Its payload, the
// anonymous role and no user, never varies, so it is marshaled once at package load.
var anonTokenClaims = &servicejsonkeys.ClaimsSignRequest{
	Usage: servicejsonkeys.KeyUsageAuth,
	Payload: lo.Must(grpcf.MarshalJSONAsAny(session.Claims{
		Roles: []string{authconfig.RoleAnon},
	})),
}

// TokenCreateAnonServiceSignClaims is the JSON Keys signing surface [TokenCreateAnon] needs.
type TokenCreateAnonServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}

// TokenCreateAnon issues an anonymous access token, carrying no user ID and only the
// anonymous role. See [session.Claims] for what such tokens grant.
type TokenCreateAnon struct {
	serviceSignClaims TokenCreateAnonServiceSignClaims
}

func NewTokenCreateAnon(serviceSignClaims TokenCreateAnonServiceSignClaims) *TokenCreateAnon {
	return &TokenCreateAnon{serviceSignClaims: serviceSignClaims}
}

func (service *TokenCreateAnon) Exec(ctx context.Context) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.TokenCreateAnon")
	defer span.End()

	accessToken, err := service.serviceSignClaims.ClaimsSign(ctx, anonTokenClaims)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue accessToken: %w", err))
	}

	return &Token{AccessToken: accessToken.GetToken()}, nil
}
