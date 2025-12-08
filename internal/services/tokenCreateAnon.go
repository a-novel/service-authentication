package services

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"google.golang.org/grpc"

	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/config"
)

var anonTokenClaims = &jkpkg.ClaimsSignRequest{
	Usage: jkpkg.KeyUsageAuth,
	Payload: lo.Must(grpcf.InterfaceToProtoAny(AccessTokenClaims{
		Roles: []string{config.RoleAnon},
	})),
}

type TokenCreateAnonSignClaimsService interface {
	ClaimsSign(
		ctx context.Context, req *jkpkg.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*jkpkg.ClaimsSignResponse, error)
}

type TokenCreateAnon struct {
	signClaimsService TokenCreateServiceSignClaims
}

func NewTokenCreateAnon(signClaimsService TokenCreateServiceSignClaims) *TokenCreateAnon {
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
