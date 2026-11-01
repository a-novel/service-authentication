package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

// ErrCredentialsCreateInvalidRegistrationData is returned when a consumed
// registration code carries malformed data or a role the service does not know.
var ErrCredentialsCreateInvalidRegistrationData = errors.New("invalid registration data")

// waitlistCleanupTimeout limits the delay from optional Google cleanup after account creation.
const waitlistCleanupTimeout = 3 * time.Second

type credentialsCreateRegistrationData struct {
	Role string `json:"role"`
}

// CredentialsCreateDao provides credential insertion capabilities.
type CredentialsCreateDao interface {
	Exec(ctx context.Context, request *dao.CredentialsInsertRequest) (*dao.Credentials, error)
}

// CredentialsCreateWaitlist removes registered addresses after the account transaction commits.
type CredentialsCreateWaitlist interface {
	Exec(ctx context.Context, request *dao.WaitlistRequest) (*dao.WaitlistResult, error)
}

// CredentialsCreateServiceShortCodeVerify checks registration short codes.
type CredentialsCreateServiceShortCodeVerify interface {
	Exec(ctx context.Context, request *ShortCodeVerifyRequest) (*ShortCode, error)
}

// CredentialsCreateServiceShortCodeConsume retires verified registration short codes.
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
	// Email is the user's email address, unique across all users.
	Email string `validate:"required,email,max=1024"`
	// Password is the plaintext password, hashed with Argon2id before storage.
	Password string `validate:"required,min=4,max=1024"`
	// ShortCode is the verification code sent to the user's email during registration.
	ShortCode string `validate:"required,max=1024"`
}

// CredentialsCreate implements user registration with email verification.
// The registration flow requires a valid short code that was previously sent
// to the user's email address to verify ownership.
type CredentialsCreate struct {
	dao                     CredentialsCreateDao
	serviceShortCodeVerify  CredentialsCreateServiceShortCodeVerify
	serviceShortCodeConsume CredentialsCreateServiceShortCodeConsume
	serviceSignClaims       CredentialsCreateServiceSignClaims
	transactor              transaction.Transactor
	waitlist                CredentialsCreateWaitlist
}

func NewCredentialsCreate(
	dao CredentialsCreateDao,
	serviceShortCodeVerify CredentialsCreateServiceShortCodeVerify,
	serviceShortCodeConsume CredentialsCreateServiceShortCodeConsume,
	serviceSignClaims CredentialsCreateServiceSignClaims,
	transactor transaction.Transactor,
	waitlist CredentialsCreateWaitlist,
) *CredentialsCreate {
	return &CredentialsCreate{
		dao:                     dao,
		serviceShortCodeVerify:  serviceShortCodeVerify,
		serviceShortCodeConsume: serviceShortCodeConsume,
		serviceSignClaims:       serviceSignClaims,
		transactor:              transactor,
		waitlist:                waitlist,
	}
}

// Exec registers a user with the request's email and password, and returns a fresh access/refresh
// token pair. A one-time registration short code gates the registration. The code check and the
// Argon2id hash run first, since a transaction holds a pooled connection. One transaction then
// retires the code and inserts the credentials, so a code retired meanwhile or a taken email
// creates no user.
//
// The committed account then leaves the invitation list within a bounded time, whatever the client
// does. A failed removal leaves the account usable, and maintenance removes the stale row.
func (service *CredentialsCreate) Exec(ctx context.Context, request *CredentialsCreateRequest) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CredentialsCreate")
	defer span.End()

	span.SetAttributes(attribute.String("credentials.email", request.Email))
	// The password and short code never go on the span. A redaction still carries its length.

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	verifyRequest := &ShortCodeVerifyRequest{
		Usage:  ShortCodeUsageRegister,
		Target: request.Email,
		Code:   request.ShortCode,
	}

	shortCode, err := service.serviceShortCodeVerify.Exec(ctx, verifyRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("verify short code: %w", err))
	}

	role, err := computeRole(shortCode.Data)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrCredentialsCreateInvalidRegistrationData))
	}

	// Hashing waits for the code, so a wrong one costs a single Argon2 run, as a missing one does.
	encryptedPassword, err := lib.GenerateArgon2(request.Password, lib.Argon2ParamsDefault)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encrypt password: %w", err))
	}

	var credentials *dao.Credentials

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		_, err := service.serviceShortCodeConsume.Exec(ctx, &ShortCodeConsumeRequest{ID: shortCode.ID})
		if err != nil {
			return fmt.Errorf("consume short code: %w", err)
		}

		daoRequest := &dao.CredentialsInsertRequest{
			ID:       uuid.New(),
			Email:    request.Email,
			Password: encryptedPassword,
			Now:      time.Now(),
			Role:     role,
		}

		credentials, err = service.dao.Exec(ctx, daoRequest)
		if err != nil {
			return fmt.Errorf("insert credentials: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	// Preserve the committed account even when Google is unavailable or the client has disconnected.
	// A failed cleanup leaves the account standing; the waitlist span records the error.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), waitlistCleanupTimeout)
	_, _ = service.waitlist.Exec(cleanupCtx, &dao.WaitlistRequest{Action: dao.WaitlistActionRemove, Email: request.Email})

	cancel()

	tokens, err := signTokenPair(ctx, service.serviceSignClaims, credentials, session.RefreshClaimsForm{})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("sign token pair: %w", err))
	}

	return tokens, nil
}

func computeRole(data []byte) (string, error) {
	if len(data) == 0 {
		return authconfig.RoleUser, nil
	}

	var registrationData credentialsCreateRegistrationData

	err := json.Unmarshal(data, &registrationData)
	if err != nil {
		return "", fmt.Errorf("decode registration data: %w", err)
	}

	_, err = authconfig.PermissionsConfigDefault.Priority(registrationData.Role)
	if err != nil {
		return "", fmt.Errorf("validate registration role: %w", err)
	}

	return registrationData.Role, nil
}
