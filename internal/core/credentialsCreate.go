package core

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// CredentialsCreateDao provides credential insertion capabilities.
type CredentialsCreateDao interface {
	Exec(ctx context.Context, request *dao.CredentialsInsertRequest) (*dao.Credentials, error)
}

// CredentialsCreateServiceShortCodeConsume validates and consumes registration short codes.
type CredentialsCreateServiceShortCodeConsume interface {
	Exec(ctx context.Context, request *ShortCodeConsumeRequest) (*ShortCode, error)
}

// CredentialsCreateServiceSignClaims provides JWT signing capabilities.
type CredentialsCreateServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
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
	dao                     CredentialsCreateDao
	serviceShortCodeConsume CredentialsCreateServiceShortCodeConsume
	serviceSignClaims       CredentialsCreateServiceSignClaims
}

func NewCredentialsCreate(
	dao CredentialsCreateDao,
	serviceShortCodeConsume CredentialsCreateServiceShortCodeConsume,
	serviceSignClaims CredentialsCreateServiceSignClaims,
) *CredentialsCreate {
	return &CredentialsCreate{
		dao:                     dao,
		serviceShortCodeConsume: serviceShortCodeConsume,
		serviceSignClaims:       serviceSignClaims,
	}
}

// Exec atomically registers a new user with the supplied email and password,
// gated by a one-time registration short code, and returns a fresh access/refresh
// token pair. The password is hashed with Argon2id before storage. If the short
// code is invalid or the credentials cannot be inserted (for example because the
// email is already taken), the transaction is rolled back and no user is created.
func (service *CredentialsCreate) Exec(ctx context.Context, request *CredentialsCreateRequest) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.CredentialsCreate")
	defer span.End()

	span.SetAttributes(attribute.String("email", request.Email))
	// Do not record the password or short code on the span, even redacted: a "*****"
	// of the same length still leaks the input length over every trace, which is
	// partial credential information an attacker reading traces could correlate.

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

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
		credentials, err = service.dao.Exec(ctx, &dao.CredentialsInsertRequest{
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

	tokens, err := signTokenPair(ctx, service.serviceSignClaims, credentials)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("sign token pair: %w", err))
	}

	return otel.ReportSuccess(span, tokens), nil
}
