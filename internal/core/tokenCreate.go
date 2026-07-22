package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// TokenCreateDao provides access to credentials lookup by email.
type TokenCreateDao interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}

// TokenCreateServiceSignClaims provides JWT signing capabilities.
type TokenCreateServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}

// TokenCreateRequest contains the credentials for user authentication.
type TokenCreateRequest struct {
	// Email is the user's email address used for authentication.
	Email string `validate:"required,email,max=1024"`
	// Password is the plaintext password to verify against the stored hash.
	Password string `validate:"required,max=1024"`
}

// TokenCreate authenticates a user by email and password and issues a fresh
// access/refresh token pair.
type TokenCreate struct {
	dao               TokenCreateDao
	serviceSignClaims TokenCreateServiceSignClaims
}

func NewTokenCreate(
	dao TokenCreateDao,
	serviceSignClaims TokenCreateServiceSignClaims,
) *TokenCreate {
	return &TokenCreate{
		dao:               dao,
		serviceSignClaims: serviceSignClaims,
	}
}

// Exec verifies the request's email and password and returns a freshly signed
// access/refresh token pair. See signTokenPair for how the two tokens are bound.
//
// It returns lib.ErrInvalidPassword when the password does not match and
// dao.ErrCredentialsSelectByEmailNotFound when the email is not registered; both
// surface as 401 at the handler.
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

	credentials, err := service.dao.Exec(ctx, &dao.CredentialsSelectByEmailRequest{
		Email: request.Email,
	})
	if err != nil {
		if errors.Is(err, dao.ErrCredentialsSelectByEmailNotFound) {
			// Burn an Argon2id verification so an unknown email costs the same as a
			// wrong password, and the latency reveals nothing about whether the email
			// is registered. Both outcomes map to 401 downstream.
			lib.DummyCompareArgon2(request.Password)
		}

		return nil, otel.ReportError(span, err)
	}

	span.SetAttributes(
		attribute.String("credentials.id", credentials.ID.String()),
		attribute.String("credentials.role", credentials.Role),
	)

	err = lib.CompareArgon2(request.Password, credentials.Password)
	if err != nil {
		// A wrong password yields lib.ErrInvalidPassword, which the handler maps to 401;
		// a malformed stored hash yields lib.ErrInvalidHash or lib.ErrIncompatibleVersion.
		// Both land on the span so it shows what the request hit.
		return nil, otel.ReportError(span, fmt.Errorf("compare password: %w", err))
	}

	tokens, err := signTokenPair(ctx, service.serviceSignClaims, credentials)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("sign token pair: %w", err))
	}

	return otel.ReportSuccess(span, tokens), nil
}
