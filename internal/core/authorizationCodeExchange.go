package core

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

var (
	// ErrAuthorizationCodeExchangeForbiddenCaller is returned by [AuthorizationCodeExchange.Exec] when
	// the caller is no client of the registry. A server that verifies no service token sees no
	// caller, and refuses too.
	ErrAuthorizationCodeExchangeForbiddenCaller = errors.New("caller is no client")
	// ErrAuthorizationCodeExchangeUnknownCode is returned by [AuthorizationCodeExchange.Exec] for a
	// code never issued, purged, or whose account no longer exists.
	ErrAuthorizationCodeExchangeUnknownCode = errors.New("unknown authorization code")
	// ErrAuthorizationCodeExchangeReused is returned by [AuthorizationCodeExchange.Exec] for a code
	// already exchanged. The exchange also ends the session the code belongs to, as RFC 9700 §4.2.4
	// advises, so the token pair the first exchange returned stops working too.
	ErrAuthorizationCodeExchangeReused = errors.New("authorization code reused")
	// ErrAuthorizationCodeExchangeExpired is returned by [AuthorizationCodeExchange.Exec] for a code
	// older than [AuthorizationCodeTTL].
	ErrAuthorizationCodeExchangeExpired = errors.New("authorization code expired")
	// ErrAuthorizationCodeExchangeForeignCode is returned by [AuthorizationCodeExchange.Exec] for a
	// code issued to another client.
	ErrAuthorizationCodeExchangeForeignCode = errors.New("authorization code issued to another client")
	// ErrAuthorizationCodeExchangeRedirectURIMismatch is returned by [AuthorizationCodeExchange.Exec]
	// when the redirect URI differs from the one the code was sent to (RFC 6749 §4.1.3).
	ErrAuthorizationCodeExchangeRedirectURIMismatch = errors.New("redirect uri is not the code's")
	// ErrAuthorizationCodeExchangeInvalidVerifier is returned by [AuthorizationCodeExchange.Exec] when
	// the S256 hash of the code verifier is not the code's challenge (RFC 7636 §4.6).
	ErrAuthorizationCodeExchangeInvalidVerifier = errors.New("invalid code verifier")
	// ErrAuthorizationCodeExchangeSessionEnded is returned by [AuthorizationCodeExchange.Exec] when the
	// session the code belongs to was signed out or has ended.
	ErrAuthorizationCodeExchangeSessionEnded = errors.New("session ended")
)

// AuthorizationCodeExchangeDaoSelect reads the code the client presents.
type AuthorizationCodeExchangeDaoSelect interface {
	Exec(ctx context.Context, request *dao.AuthorizationCodeSelectRequest) (*dao.AuthorizationCode, error)
}

// AuthorizationCodeExchangeDaoConsume marks the code exchanged, once.
type AuthorizationCodeExchangeDaoConsume interface {
	Exec(ctx context.Context, request *dao.AuthorizationCodeConsumeRequest) (*dao.AuthorizationCode, error)
}

// AuthorizationCodeExchangeDaoCredentials loads the account the code signs in.
type AuthorizationCodeExchangeDaoCredentials interface {
	Exec(ctx context.Context, request *dao.CredentialsSelectRequest) (*dao.Credentials, error)
}

// AuthorizationCodeExchangeDaoRevocationExist reports whether the code's session was signed out.
type AuthorizationCodeExchangeDaoRevocationExist interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationExistRequest) (bool, error)
}

// AuthorizationCodeExchangeDaoRevocationInsert ends the session of a reused code.
type AuthorizationCodeExchangeDaoRevocationInsert interface {
	Exec(ctx context.Context, request *dao.RefreshTokenRevocationInsertRequest) error
}

// AuthorizationCodeExchangeServiceSignClaims signs the token pair.
type AuthorizationCodeExchangeServiceSignClaims interface {
	ClaimsSign(
		ctx context.Context, req *servicejsonkeys.ClaimsSignRequest, opts ...grpc.CallOption,
	) (*servicejsonkeys.ClaimsSignResponse, error)
}

// AuthorizationCodeExchangeRequest carries a client's code exchange (RFC 6749 §4.1.3).
type AuthorizationCodeExchangeRequest struct {
	// Caller is the verified Kubernetes caller of the request, which names the client. It is empty
	// when the server verifies none.
	Caller string

	// Code is the code parameter, as [AuthorizationCodeCreate] issued it.
	Code string `validate:"required,len=43"`
	// CodeVerifier is the code_verifier parameter (RFC 7636 §4.5).
	CodeVerifier string `validate:"required,min=43,max=128"`
	// RedirectURI is the redirect_uri parameter, the one the code was sent to.
	RedirectURI string `validate:"required,max=2048"`
}

// AuthorizationCodeExchange trades an authorization code for a token pair in the session the code
// belongs to, on behalf of the client the code was issued to. Only that client's caller may
// exchange it, once, with the redirect URI and the code verifier the code is bound to. The account
// must exist: an exchange signs in, and never creates an account.
//
// A refused exchange leaves the code as it was, except a reuse, which ends the session. The pair is
// signed before the code is consumed, so a failed signature leaves the code to a retry. The
// consumption is one conditional write: of two concurrent exchanges, the second finds the code
// consumed and ends the session as any reuse does.
type AuthorizationCodeExchange struct {
	daoSelect           AuthorizationCodeExchangeDaoSelect
	daoConsume          AuthorizationCodeExchangeDaoConsume
	daoCredentials      AuthorizationCodeExchangeDaoCredentials
	daoRevocationExist  AuthorizationCodeExchangeDaoRevocationExist
	daoRevocationInsert AuthorizationCodeExchangeDaoRevocationInsert
	serviceSignClaims   AuthorizationCodeExchangeServiceSignClaims
	oauth               authconfig.OAuth
	// refreshTokenLeeway is the clock-skew tolerance the verifier grants an expired refresh token.
	refreshTokenLeeway time.Duration
}

func NewAuthorizationCodeExchange(
	daoSelect AuthorizationCodeExchangeDaoSelect,
	daoConsume AuthorizationCodeExchangeDaoConsume,
	daoCredentials AuthorizationCodeExchangeDaoCredentials,
	daoRevocationExist AuthorizationCodeExchangeDaoRevocationExist,
	daoRevocationInsert AuthorizationCodeExchangeDaoRevocationInsert,
	serviceSignClaims AuthorizationCodeExchangeServiceSignClaims,
	oauth authconfig.OAuth,
	refreshTokenLeeway time.Duration,
) *AuthorizationCodeExchange {
	return &AuthorizationCodeExchange{
		daoSelect:           daoSelect,
		daoConsume:          daoConsume,
		daoCredentials:      daoCredentials,
		daoRevocationExist:  daoRevocationExist,
		daoRevocationInsert: daoRevocationInsert,
		serviceSignClaims:   serviceSignClaims,
		oauth:               oauth,
		refreshTokenLeeway:  refreshTokenLeeway,
	}
}

func (service *AuthorizationCodeExchange) Exec(
	ctx context.Context, request *AuthorizationCodeExchangeRequest,
) (*Token, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.AuthorizationCodeExchange")
	defer span.End()

	clientID, ok := service.clientOf(request.Caller)
	if !ok {
		return nil, refuseCodeExchange(span, ErrAuthorizationCodeExchangeForbiddenCaller, nil)
	}

	span.SetAttributes(attribute.String("client.id", clientID))

	err := validate.Struct(request)
	if err != nil {
		return nil, refuseCodeExchange(span, ErrInvalidRequest, err)
	}

	now := time.Now()
	id := sha256.Sum256([]byte(request.Code))

	code, err := service.daoSelect.Exec(ctx, &dao.AuthorizationCodeSelectRequest{ID: id[:]})
	if errors.Is(err, dao.ErrAuthorizationCodeSelectNotFound) {
		return nil, refuseCodeExchange(span, ErrAuthorizationCodeExchangeUnknownCode, err)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select authorization code: %w", err))
	}

	span.SetAttributes(
		attribute.String("user.id", code.UserID.String()),
		attribute.String("session.id", code.SessionID),
	)

	if code.ConsumedAt != nil {
		return nil, service.refuseReuse(ctx, span, code, now)
	}

	err = checkCodeExchange(code, clientID, request, now)
	if err != nil {
		return nil, refuseCodeExchange(span, err, nil)
	}

	revocationRequest := &dao.RefreshTokenRevocationExistRequest{JTIs: []string{code.SessionID}}

	revoked, err := service.daoRevocationExist.Exec(ctx, revocationRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("check session revocation: %w", err))
	}

	if revoked {
		return nil, refuseCodeExchange(span, ErrAuthorizationCodeExchangeSessionEnded, nil)
	}

	credentials, err := service.daoCredentials.Exec(ctx, &dao.CredentialsSelectRequest{ID: code.UserID})
	if errors.Is(err, dao.ErrCredentialsSelectNotFound) {
		return nil, refuseCodeExchange(span, ErrAuthorizationCodeExchangeUnknownCode, err)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("select credentials: %w", err))
	}

	joined := session.RefreshClaimsForm{SessionID: code.SessionID, SessionExp: code.SessionExpiresAt.Unix()}

	tokens, err := signTokenPair(ctx, service.serviceSignClaims, credentials, joined)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("sign token pair: %w", err))
	}

	_, err = service.daoConsume.Exec(ctx, &dao.AuthorizationCodeConsumeRequest{ID: id[:], Now: now})
	if errors.Is(err, dao.ErrAuthorizationCodeConsumeNotFound) {
		// Another exchange consumed the code since it was read, and the pair signed here is dropped.
		return nil, service.refuseReuse(ctx, span, code, now)
	}

	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("consume authorization code: %w", err))
	}

	return tokens, nil
}

// clientOf returns the ID of the client whose caller is caller. No caller names no client.
func (service *AuthorizationCodeExchange) clientOf(caller string) (string, bool) {
	if caller == "" {
		return "", false
	}

	for clientID, client := range service.oauth.Clients {
		if client.Caller == caller {
			return clientID, true
		}
	}

	return "", false
}

// refuseReuse ends the session of a reused code, which its first exchange extended to the client,
// and returns the refusal it reports on parent. The revocation lasts as long as a refresh token of
// the session may.
func (service *AuthorizationCodeExchange) refuseReuse(
	ctx context.Context, parent trace.Span, code *dao.AuthorizationCode, now time.Time,
) error {
	ctx, span := otel.Tracer().Start(ctx, "core.AuthorizationCodeExchange(refuseReuse)")
	defer span.End()

	revocation := &dao.RefreshTokenRevocationInsertRequest{
		JTIs:      []string{code.SessionID},
		ExpiresAt: code.SessionExpiresAt.Add(service.refreshTokenLeeway),
		Now:       now,
	}

	err := service.daoRevocationInsert.Exec(ctx, revocation)
	if err != nil {
		err = otel.ReportError(span, fmt.Errorf("revoke session: %w", err))

		return otel.ReportError(parent, err)
	}

	return refuseCodeExchange(parent, ErrAuthorizationCodeExchangeReused, nil)
}

// checkCodeExchange returns the refusal for a code that expired, belongs to another client or
// redirect URI, does not match the verifier, or whose session ended.
func checkCodeExchange(
	code *dao.AuthorizationCode, clientID string, request *AuthorizationCodeExchangeRequest, now time.Time,
) error {
	challenge := sha256.Sum256([]byte(request.CodeVerifier))
	verified := subtle.ConstantTimeCompare(
		[]byte(base64.RawURLEncoding.EncodeToString(challenge[:])), []byte(code.CodeChallenge),
	)

	switch {
	case !now.Before(code.ExpiresAt):
		return ErrAuthorizationCodeExchangeExpired
	case code.ClientID != clientID:
		return ErrAuthorizationCodeExchangeForeignCode
	case code.RedirectURI != request.RedirectURI:
		return ErrAuthorizationCodeExchangeRedirectURIMismatch
	case verified != 1:
		return ErrAuthorizationCodeExchangeInvalidVerifier
	case !now.Before(code.SessionExpiresAt):
		return ErrAuthorizationCodeExchangeSessionEnded
	}

	return nil
}

// refuseCodeExchange reports reason on span, joined onto cause when a failed step caused the
// refusal. The span's authorization_code.refusal attribute names the reason, which the Services
// tab counts per client.
func refuseCodeExchange(span trace.Span, reason, cause error) error {
	span.SetAttributes(attribute.String("authorization_code.refusal", reason.Error()))

	return otel.ReportError(span, errors.Join(cause, reason))
}
