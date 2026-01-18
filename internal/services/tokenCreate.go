package services

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel-kit/golib/grpcf"
	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// TokenCreateRepository provides access to credentials lookup by email.
type TokenCreateRepository interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectByEmailRequest) (*dao.Credentials, error)
}

// TokenCreateServiceSignClaims provides JWT signing capabilities.
type TokenCreateServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *jkpkg.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*jkpkg.ClaimsSignResponse, error)
}

// TokenCreateRequest contains the credentials for user authentication.
type TokenCreateRequest struct {
	// Email is the user's email address used for authentication.
	Email string `validate:"required,email,max=1024"`
	// Password is the plaintext password to verify against the stored hash.
	Password string `validate:"required,max=1024"`
}

// TokenCreate implements the authentication service using a two-token system.
// It issues both an access token (short-lived, for API authorization) and
// a refresh token (long-lived, for obtaining new access tokens).
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

// Exec authenticates a user and returns a token pair.
//
// The two-token system works as follows:
//   - Access Token: Short-lived JWT containing user ID, roles, and a reference to the refresh token.
//     Used for API authorization via the Authorization header.
//   - Refresh Token: Long-lived JWT used to obtain new access tokens without re-authentication.
//     Contains only the user ID and is bound to the access token via its JTI.
//
// Returns ErrInvalidPassword if the password doesn't match, or dao.ErrCredentialsSelectByEmailNotFound
// if the email is not registered.
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
	err = lib.CompareArgon2(request.Password, credentials.Password)
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
