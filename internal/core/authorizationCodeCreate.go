package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

const (
	// AuthorizationCodeTTL is how long a code exchanges after it is issued. RFC 6749 §4.1.2 asks for a
	// short lifetime, and the client exchanges the code within the redirect that delivers it.
	AuthorizationCodeTTL = time.Minute
	// AuthorizationCodeSize is the number of random bytes in a code. Its 256 bits keep a guess far
	// below the 2^-160 chance RFC 6749 §10.10 asks for.
	AuthorizationCodeSize = 32
	// CodeChallengeMethodS256 is the only PKCE method a code accepts. RFC 9700 §2.1.1 names it the only
	// method that keeps the code verifier out of the authorization request.
	CodeChallengeMethodS256 = "S256"
)

// ErrAuthorizationCodeCreateSessionEnded is returned by [AuthorizationCodeCreate.Exec] when the
// session of the access token has ended, though the access token has not expired yet. The account
// platform then signs the person in again.
var ErrAuthorizationCodeCreateSessionEnded = errors.New("session ended")

// AuthorizationCodeCreateDao stores the new code.
type AuthorizationCodeCreateDao interface {
	Exec(ctx context.Context, request *dao.AuthorizationCodeInsertRequest) error
}

// AuthorizationCodeCreateRequest asks for a code that answers an authorization request, for a person
// signed in on the account platform.
type AuthorizationCodeCreateRequest struct {
	// Caller is the verified Kubernetes caller of the request, empty when the server verifies none.
	Caller string

	// UserID is the person signing in, taken from their verified access token.
	UserID uuid.UUID `validate:"required"`
	// SessionID is the session of that access token, which the client's token pair joins.
	SessionID string `validate:"required,max=1024"`
	// SessionExp is when that session ends, as a Unix timestamp in seconds. An access token signed
	// before sessions carried their end lacks it, and reads as a session that ended.
	SessionExp int64

	// Params are the parameters of the authorization request, as [AuthorizationRequestValidate] reads
	// them.
	Params url.Values
}

// AuthorizationCode is a code issued to a client, for the account platform to send to its redirect
// URI.
type AuthorizationCode struct {
	// Code is 256 random bits, base64url-encoded. Only its SHA-256 is stored.
	Code string
	// RedirectURI is where the account platform sends the code, with the request's state and the
	// account platform's iss (RFC 9207).
	RedirectURI string
}

// AuthorizationCodeCreate issues an authorization code for a person signed in on the account
// platform (RFC 6749 §4.1.2). Only the account platform may ask, for a request that
// [AuthorizationRequestValidate] accepts, and the request is checked again. The code is bound to
// the person, their session, the client, its redirect URI and the PKCE challenge and method
// (RFC 7636 §4.4), and [TokenGrant] trades it once, within [AuthorizationCodeTTL].
type AuthorizationCodeCreate struct {
	dao   AuthorizationCodeCreateDao
	oauth authconfig.OAuth
}

func NewAuthorizationCodeCreate(dao AuthorizationCodeCreateDao, oauth authconfig.OAuth) *AuthorizationCodeCreate {
	return &AuthorizationCodeCreate{dao: dao, oauth: oauth}
}

func (service *AuthorizationCodeCreate) Exec(
	ctx context.Context, request *AuthorizationCodeCreateRequest,
) (*AuthorizationCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.AuthorizationCodeCreate")
	defer span.End()

	span.SetAttributes(
		attribute.String("client.id", request.Params.Get("client_id")),
		attribute.String("user.id", request.UserID.String()),
		attribute.String("session.id", request.SessionID),
	)

	err := checkAccountPlatform(service.oauth, request.Caller)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	err = validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	authorization, err := parseAuthorizationRequest(service.oauth.Clients, request.Params)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	now := time.Now()
	sessionExpiresAt := time.Unix(request.SessionExp, 0)

	if request.SessionExp == 0 || !now.Before(sessionExpiresAt) {
		return nil, otel.ReportError(span, ErrAuthorizationCodeCreateSessionEnded)
	}

	secret := make([]byte, AuthorizationCodeSize)
	// Read never fails: the program crashes first.
	_, _ = rand.Read(secret)

	code := base64.RawURLEncoding.EncodeToString(secret)
	id := sha256.Sum256([]byte(code))

	insertRequest := &dao.AuthorizationCodeInsertRequest{
		ID:                   id[:],
		UserID:               request.UserID,
		ClientID:             authorization.ClientID,
		RedirectURI:          authorization.RedirectURI,
		RedirectURIInRequest: authorization.RedirectURIInRequest,
		CodeChallenge:        authorization.CodeChallenge,
		CodeChallengeMethod:  authorization.CodeChallengeMethod,
		SessionID:            request.SessionID,
		SessionExpiresAt:     sessionExpiresAt,
		ExpiresAt:            now.Add(AuthorizationCodeTTL),
	}

	err = service.dao.Exec(ctx, insertRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("insert authorization code: %w", err))
	}

	return &AuthorizationCode{Code: code, RedirectURI: authorization.RedirectURI}, nil
}
