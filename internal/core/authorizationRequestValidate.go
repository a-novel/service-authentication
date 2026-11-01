package core

import (
	"context"
	"net/url"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

// AuthorizationRequestValidateRequest carries an authorization request, as the account platform
// received it on its authorization endpoint.
type AuthorizationRequestValidateRequest struct {
	// Caller is the verified Kubernetes caller of the request, empty when the server verifies none.
	Caller string
	// Params are the parameters of the authorization request, such as client_id and code_challenge.
	Params url.Values
}

// AuthorizationRequestValidate checks an authorization request for the account platform before it
// signs the person in, as RFC 6749 §4.1.1 and RFC 7636 §4.3 define the request. Only the account
// platform may ask. A refusal is an [OAuthError]: its RedirectURI says whether the account
// platform sends it to the client, or shows it to the person.
type AuthorizationRequestValidate struct {
	oauth authconfig.OAuth
}

func NewAuthorizationRequestValidate(oauth authconfig.OAuth) *AuthorizationRequestValidate {
	return &AuthorizationRequestValidate{oauth: oauth}
}

func (service *AuthorizationRequestValidate) Exec(
	ctx context.Context, request *AuthorizationRequestValidateRequest,
) (*AuthorizationRequest, error) {
	_, span := otel.Tracer().Start(ctx, "core.AuthorizationRequestValidate")
	defer span.End()

	err := checkAccountPlatform(service.oauth, request.Caller)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	setClientAttribute(span, service.oauth.Clients, request.Params)

	authorization, err := parseAuthorizationRequest(service.oauth.Clients, request.Params)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return authorization, nil
}
