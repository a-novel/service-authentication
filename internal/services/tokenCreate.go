package services

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	"github.com/a-novel/golib/grpcf"
	"github.com/a-novel/golib/otel"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

type TokenCreateRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}
type TokenCreateServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *jkpkg.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*jkpkg.ClaimsSignResponse, error)
}

type TokenCreateRequest struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,max=1024"`
}

type TokenCreate struct {
	repository        TokenCreateRepository
	serviceSignClaims TokenCreateServiceSignClaims
}

func NewTokenCreate(
	repository TokenCreateRepository,
	serviceSignClaims TokenCreateServiceSignClaims,
) *TokenCreate {
	return &TokenCreate{
		repository:        repository,
		serviceSignClaims: serviceSignClaims,
	}
}

func (service *TokenCreate) Exec(
	ctx context.Context, request *TokenCreateRequest,
) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.TokenCreate")
	defer span.End()

	span.SetAttributes(attribute.String("email", request.Email))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Retrieve credentials.
	credentials, err := service.repository.Exec(ctx, &dao.CredentialsSelectByEmailRequest{
		Email: request.Email,
	})
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(
		attribute.String("credentials.id", credentials.ID.String()),
		attribute.String("credentials.role", credentials.Role),
	)

	// Validate password.
	err = lib.CompareScrypt(request.Password, credentials.Password)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("compare password: %w", err))
	}

	refreshTokenPayload, err := grpcf.InterfaceToProtoAny(RefreshTokenClaimsForm{
		UserID: credentials.ID,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("grpcf marshal: %w", err))
	}

	refreshToken, err := service.serviceSignClaims.ClaimsSign(
		ctx,
		&jkpkg.ClaimsSignRequest{
			Usage:   jkpkg.KeyUsageAuthRefresh,
			Payload: refreshTokenPayload,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue refresh token: %w", err))
	}

	refreshTokenRecipient := jwt.NewRecipient(jwt.RecipientConfig{
		Plugins: []jwt.RecipientPlugin{jws.NewInsecureVerifier()},
	})

	var refreshTokenClaims RefreshTokenClaims

	err = refreshTokenRecipient.Consume(ctx, refreshToken.GetToken(), &refreshTokenClaims)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("unmarshal refresh token claims: %w", err))
	}

	// Generate a new authentication token.
	accessTokenPayload, err := grpcf.InterfaceToProtoAny(AccessTokenClaims{
		UserID:         &credentials.ID,
		Roles:          []string{credentials.Role},
		RefreshTokenID: refreshTokenClaims.Jti,
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("grpcf marshal: %w", err))
	}

	// Generate a new authentication token.
	accessToken, err := service.serviceSignClaims.ClaimsSign(
		ctx,
		&jkpkg.ClaimsSignRequest{
			Usage:   jkpkg.KeyUsageAuth,
			Payload: accessTokenPayload,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue accessToken: %w", err))
	}

	return otel.ReportSuccess(span, &Token{
		AccessToken:  accessToken.GetToken(),
		RefreshToken: refreshToken.GetToken(),
	}), nil
}
