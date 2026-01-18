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

// CredentialsCreateRepository provides credential insertion capabilities.
type CredentialsCreateRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsInsertRequest) (*dao.Credentials, error)
}

// CredentialsCreateServiceShortCodeConsume validates and consumes registration short codes.
type CredentialsCreateServiceShortCodeConsume interface {
	Exec(ctx context.Context, request *ShortCodeConsumeRequest) (*ShortCode, error)
}

// CredentialsCreateServiceSignClaims provides JWT signing capabilities.
type CredentialsCreateServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *jkpkg.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*jkpkg.ClaimsSignResponse, error)
}

// CredentialsCreateRequest contains the data required to register a new user.
type CredentialsCreateRequest struct {
	// Email is the user's email address, must be unique across all users.
	Email string `validate:"required,email,max=1024"`
	// Password is the plaintext password, will be hashed using Argon2id before storage.
	// Must be at least 4 characters.
	Password string `validate:"required,min=4,max=1024"`
	// ShortCode is the verification code sent to the user's email during registration.
	ShortCode string `validate:"required,max=1024"`
}

// CredentialsCreate implements user registration with email verification.
// The registration flow requires a valid short code that was previously sent
// to the user's email address to verify ownership.
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

// Exec registers a new user and returns authentication tokens.
//
// The registration process runs within a database transaction to ensure atomicity:
//  1. Validates the short code matches the target email
//  2. Consumes (deletes) the short code to prevent reuse
//  3. Hashes the password using Argon2id
//  4. Inserts the new credentials with the default user role
//  5. Generates and returns access and refresh tokens
//
// If any step fails, the transaction is rolled back and no user is created.
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
	encryptedPassword, err := lib.GenerateArgon2(request.Password, lib.Argon2ParamsDefault)
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
