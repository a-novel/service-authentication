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

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"
	jkmodels "github.com/a-novel/service-json-keys/models"

	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
)

type CredentialsCreateRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsInsertRequest) (*dao.Credentials, error)
}
type CredentialsCreateServiceShortCodeConsume interface {
	Exec(ctx context.Context, request *ShortCodeConsumeRequest) (*ShortCode, error)
}
type CredentialsCreateServiceSignClaims interface {
	SignClaims(ctx context.Context, usage jkmodels.KeyUsage, claims any) (string, error)
}

type CredentialsCreateRequest struct {
	Email     string `validate:"required,email"`
	Password  string `validate:"required_without=ShortCode,max=1024"`
	ShortCode string `validate:"required_without=Password,max=1024"`
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

	refreshToken, err := service.serviceSignClaims.SignClaims(
		ctx,
		jkmodels.KeyUsageRefresh,
		RefreshTokenClaimsForm{
			UserID: credentials.ID,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue refresh token: %w", err))
	}

	refreshTokenRecipient := jwt.NewRecipient(jwt.RecipientConfig{
		Plugins: []jwt.RecipientPlugin{jws.NewInsecureVerifier()},
	})

	var refreshTokenClaims RefreshTokenClaims

	err = refreshTokenRecipient.Consume(ctx, refreshToken, &refreshTokenClaims)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("unmarshal refresh token claims: %w", err))
	}

	// Generate a new authentication token.
	accessToken, err := service.serviceSignClaims.SignClaims(
		ctx,
		jkmodels.KeyUsageAuth,
		AccessTokenClaims{
			UserID:         &credentials.ID,
			Roles:          []string{credentials.Role},
			RefreshTokenID: refreshTokenClaims.Jti,
		},
	)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("issue accessToken: %w", err))
	}

	return otel.ReportSuccess(span, &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}), nil
}
