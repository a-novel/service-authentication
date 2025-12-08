package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

type CredentialsCreateRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsInsertRequest) (*dao.Credentials, error)
}
type CredentialsCreateServiceShortCodeConsume interface {
	Exec(ctx context.Context, request *ShortCodeConsumeRequest) (*ShortCode, error)
}
type CredentialsCreateServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *jkpkg.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*jkpkg.ClaimsSignResponse, error)
}

type CredentialsCreateRequest struct {
	Email     string `validate:"required,email"`
	Password  string `validate:"required,max=1024"`
	ShortCode string `validate:"required,max=1024"`
}

type CredentialsCreate struct {
	repository              CredentialsCreateRepository
	serviceShortCodeConsume CredentialsCreateServiceShortCodeConsume
	serviceSignClaims       CredentialsCreateServiceSignClaims
}

func NewCredentialsCreate(
	repository CredentialsCreateRepository,
	serviceShortCodeConsume CredentialsCreateServiceShortCodeConsume,
	serviceSignClaims CredentialsCreateServiceSignClaims,
) *CredentialsCreate {
	return &CredentialsCreate{
		repository:              repository,
		serviceShortCodeConsume: serviceShortCodeConsume,
		serviceSignClaims:       serviceSignClaims,
	}
}

func (service *CredentialsCreate) Exec(ctx context.Context, request *CredentialsCreateRequest) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsCreate")
	defer span.End()

	span.SetAttributes(attribute.String("email", request.Email))
	span.SetAttributes(attribute.String("password", strings.Repeat("*", len(request.Password))))
	span.SetAttributes(attribute.String("shortCode", strings.Repeat("*", len(request.ShortCode))))

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	// Encrypt the password.
	encryptedPassword, err := lib.GenerateScrypt(request.Password, lib.ScryptParamsDefault)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt password: %w", err))
	}

	var credentials *dao.Credentials

	err = postgres.RunInTx(ctx, nil, func(ctx context.Context, tx bun.IDB) error {
		// Verify short code.
		_, err = service.serviceShortCodeConsume.Exec(ctx, &ShortCodeConsumeRequest{
			Usage:  ShortCodeUsageRegister,
			Target: request.Email,
			Code:   request.ShortCode,
		})
		if err != nil {
			return fmt.Errorf("consume short code: %w", err)
		}

		// Create credentials.
		credentials, err = service.repository.Exec(ctx, &dao.CredentialsInsertRequest{
			ID:       uuid.New(),
			Email:    request.Email,
			Password: encryptedPassword,
			Now:      time.Now(),
			Role:     config.RoleUser,
		})
		if err != nil {
			return fmt.Errorf("insert credentials: %w", err)
		}

		span.SetAttributes(attribute.String("credentials.id", credentials.ID.String()))

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
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

	accessTokenPayload, err := grpcf.InterfaceToProtoAny(AccessTokenClaims{
		UserID:         &credentials.ID,
		Roles:          []string{credentials.Role},
		RefreshTokenID: refreshTokenClaims.Jti,
	})

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
