package core

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

// ErrAuthorizationRequestForbiddenCaller is returned by [AuthorizationRequestValidate.Exec] and
// [AuthorizationCodeCreate.Exec] when the caller is not the account platform. A server that
// verifies no service token sees no caller, and refuses too.
var ErrAuthorizationRequestForbiddenCaller = errors.New("caller is not the account platform")

// The refusals of an authorization request that the account platform shows the person, without
// redirecting anywhere (RFC 6749 §4.1.2.1).
var (
	// ErrAuthorizationRequestUnknownClient is returned by [AuthorizationRequestValidate.Exec] and
	// [AuthorizationCodeCreate.Exec] for a client_id that is missing, repeated or unregistered.
	ErrAuthorizationRequestUnknownClient = &OAuthError{Code: OAuthErrorInvalidRequest, Description: "unknown client"}
	// ErrAuthorizationRequestInvalidRedirectURI is returned by [AuthorizationRequestValidate.Exec] and
	// [AuthorizationCodeCreate.Exec] for a redirect_uri that is repeated, or other than the client's
	// one as an exact string (RFC 9700 §4.1.3).
	ErrAuthorizationRequestInvalidRedirectURI = &OAuthError{
		Code: OAuthErrorInvalidRequest, Description: "redirect uri is not the client's",
	}
)

// The refusals of an authorization request that the account platform redirects to the client
// (RFC 6749 §4.1.2.1), as the RedirectURI of the returned [OAuthError] names.
var (
	// ErrAuthorizationRequestNoResponseType is returned for a request without response_type.
	ErrAuthorizationRequestNoResponseType = &OAuthError{
		Code: OAuthErrorInvalidRequest, Description: "response_type is required",
	}
	// ErrAuthorizationRequestUnsupportedResponseType is returned for a response_type other than code.
	ErrAuthorizationRequestUnsupportedResponseType = &OAuthError{
		Code: OAuthErrorUnsupportedResponseType, Description: "response_type must be code",
	}
	// ErrAuthorizationRequestScope is returned for a request that names a scope.
	ErrAuthorizationRequestScope = &OAuthError{Code: OAuthErrorInvalidScope, Description: "no scope is granted"}
	// ErrAuthorizationRequestNoCodeChallenge is returned for a request without code_challenge
	// (RFC 7636 §4.4.1).
	ErrAuthorizationRequestNoCodeChallenge = &OAuthError{
		Code: OAuthErrorInvalidRequest, Description: "code challenge required",
	}
	// ErrAuthorizationRequestUnsupportedTransform is returned for a code_challenge_method other than
	// S256. A request without one asks for plain, which the server does not support (RFC 7636 §4.3,
	// §4.4.1).
	ErrAuthorizationRequestUnsupportedTransform = &OAuthError{
		Code: OAuthErrorInvalidRequest, Description: "transform algorithm not supported",
	}
	// ErrAuthorizationRequestInvalidCodeChallenge is returned for a code_challenge that is not the
	// base64url encoding of a SHA-256 digest (RFC 7636 §4.2).
	ErrAuthorizationRequestInvalidCodeChallenge = &OAuthError{
		Code: OAuthErrorInvalidRequest, Description: "code challenge malformed",
	}
)

// s256Challenge matches the S256 transform of a code verifier: a SHA-256 digest in base64url without
// padding (RFC 7636 §4.2).
var s256Challenge = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

// authorizationRequestParams are the parameters of an authorization request (RFC 6749 §4.1.1,
// RFC 7636 §4.3). The server ignores any other.
var authorizationRequestParams = []string{
	"response_type", "client_id", "redirect_uri", "scope", "state", "code_challenge", "code_challenge_method",
}

// AuthorizationRequestValidateRequest carries an authorization request, as the account platform
// received it on its authorization endpoint.
type AuthorizationRequestValidateRequest struct {
	// Caller is the verified Kubernetes caller of the request, empty when the server verifies none.
	Caller string
	// Params are the parameters of the authorization request, such as client_id and code_challenge.
	Params url.Values
}

// AuthorizationRequest is an authorization request the server accepts.
type AuthorizationRequest struct {
	ClientID string
	// RedirectURI is where the account platform sends the response: the URI the request named, or
	// the client's one when it named none (RFC 6749 §3.1.2.3).
	RedirectURI string
	// RedirectURIInRequest reports whether the request named the redirect URI. The token request
	// must then name it too (RFC 6749 §4.1.3).
	RedirectURIInRequest bool
	// CodeChallenge is the PKCE challenge, which the code verifier must match at the token endpoint.
	CodeChallenge string
	// CodeChallengeMethod is the PKCE transform, [CodeChallengeMethodS256].
	CodeChallengeMethod string
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

	span.SetAttributes(attribute.String("client.id", request.Params.Get("client_id")))

	err := checkAccountPlatform(service.oauth, request.Caller)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	authorization, err := parseAuthorizationRequest(service.oauth.Clients, request.Params)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	return authorization, nil
}

// checkAccountPlatform refuses any caller but the account platform, and no caller.
func checkAccountPlatform(oauth authconfig.OAuth, caller string) error {
	if caller == "" || caller != oauth.AccountPlatformCaller {
		return fmt.Errorf("%w: %q", ErrAuthorizationRequestForbiddenCaller, caller)
	}

	return nil
}

// parseAuthorizationRequest reads the authorization request params carry, against the client
// registry. It first checks the client and its redirect URI, whose refusals redirect nowhere, then
// the other parameters, whose refusals redirect to the client (RFC 6749 §4.1.2.1).
func parseAuthorizationRequest(clients map[string]authconfig.Client, params url.Values) (*AuthorizationRequest, error) {
	clientID, ok := oauthParam(params, "client_id")
	client, registered := clients[clientID]

	if !ok || !registered {
		return nil, ErrAuthorizationRequestUnknownClient
	}

	redirectURI, ok := oauthParam(params, "redirect_uri")
	if !ok || (redirectURI != "" && redirectURI != client.RedirectURI) {
		return nil, ErrAuthorizationRequestInvalidRedirectURI
	}

	authorization := &AuthorizationRequest{
		ClientID:             clientID,
		RedirectURI:          client.RedirectURI,
		RedirectURIInRequest: redirectURI != "",
		CodeChallenge:        params.Get("code_challenge"),
		CodeChallengeMethod:  params.Get("code_challenge_method"),
	}

	err := checkAuthorizationParams(authorization, params)
	if err != nil {
		return nil, err.redirectTo(client.RedirectURI)
	}

	return authorization, nil
}

// checkAuthorizationParams returns the refusal of an authorization request whose client and
// redirect URI hold, or nil.
func checkAuthorizationParams(authorization *AuthorizationRequest, params url.Values) *OAuthError {
	responseType := params.Get("response_type")

	switch {
	case oauthParamsRepeated(params, authorizationRequestParams...):
		return ErrOAuthRepeatedParameter
	case responseType == "":
		return ErrAuthorizationRequestNoResponseType
	case responseType != "code":
		return ErrAuthorizationRequestUnsupportedResponseType
	case params.Get("scope") != "":
		return ErrAuthorizationRequestScope
	case authorization.CodeChallenge == "":
		return ErrAuthorizationRequestNoCodeChallenge
	case authorization.CodeChallengeMethod != CodeChallengeMethodS256:
		return ErrAuthorizationRequestUnsupportedTransform
	case !s256Challenge.MatchString(authorization.CodeChallenge):
		return ErrAuthorizationRequestInvalidCodeChallenge
	}

	return nil
}
