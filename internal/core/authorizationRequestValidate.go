package core

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

var (
	// ErrAuthorizationRequestForbiddenCaller is returned by [AuthorizationRequestValidate.Exec] and
	// [AuthorizationCodeCreate.Exec] when the caller is not the account platform. A server that
	// verifies no service token sees no caller, and refuses too.
	ErrAuthorizationRequestForbiddenCaller = errors.New("caller is not the account platform")
	// ErrAuthorizationRequestUnknownClient is returned by [AuthorizationRequestValidate.Exec] and
	// [AuthorizationCodeCreate.Exec] when the client registry holds no client with the requested ID.
	ErrAuthorizationRequestUnknownClient = errors.New("unknown client")
	// ErrAuthorizationRequestRedirectURIMismatch is returned by [AuthorizationRequestValidate.Exec]
	// and [AuthorizationCodeCreate.Exec] when the redirect URI is not exactly the client's.
	ErrAuthorizationRequestRedirectURIMismatch = errors.New("redirect uri is not the client's")
)

// AuthorizationRequestValidateRequest names the client of an authorization request.
type AuthorizationRequestValidateRequest struct {
	// Caller is the verified Kubernetes caller of the request, empty when the server verifies none.
	Caller string

	// ClientID is the client_id parameter of the authorization request (RFC 6749 §4.1.1).
	ClientID string `validate:"required,max=256"`
	// RedirectURI is its redirect_uri parameter.
	RedirectURI string `validate:"required,max=2048"`
}

// AuthorizationRequestValidate tells the account platform whether an authorization request names a
// registered client and that client's redirect URI. Until it does, the account platform shows an
// error page and redirects nowhere (RFC 6749 §4.1.2.1). Only the account platform may ask.
type AuthorizationRequestValidate struct {
	oauth authconfig.OAuth
}

func NewAuthorizationRequestValidate(oauth authconfig.OAuth) *AuthorizationRequestValidate {
	return &AuthorizationRequestValidate{oauth: oauth}
}

func (service *AuthorizationRequestValidate) Exec(
	ctx context.Context, request *AuthorizationRequestValidateRequest,
) error {
	_, span := otel.Tracer().Start(ctx, "core.AuthorizationRequestValidate")
	defer span.End()

	span.SetAttributes(attribute.String("client.id", request.ClientID))

	err := validate.Struct(request)
	if err != nil {
		return otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	err = checkAuthorizationRequest(service.oauth, request.Caller, request.ClientID, request.RedirectURI)
	if err != nil {
		return otel.ReportError(span, err)
	}

	return nil
}

// checkAuthorizationRequest refuses any caller but the account platform, and a client ID and
// redirect URI that the client registry does not hold together. The redirect URI must equal the
// client's as a string, as RFC 9700 §4.1.3 requires: a prefix, another path, a query or another host
// differs.
func checkAuthorizationRequest(oauth authconfig.OAuth, caller, clientID, redirectURI string) error {
	if caller == "" || caller != oauth.AccountPlatformCaller {
		return fmt.Errorf("%w: %q", ErrAuthorizationRequestForbiddenCaller, caller)
	}

	client, ok := oauth.Clients[clientID]
	if !ok {
		return ErrAuthorizationRequestUnknownClient
	}

	if redirectURI != client.RedirectURI {
		return ErrAuthorizationRequestRedirectURIMismatch
	}

	return nil
}
