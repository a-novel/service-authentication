package core

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

// CodeChallengeMethodS256 is the only PKCE method a code accepts. RFC 9700 §2.1.1 names it the only
// method that keeps the code verifier out of the authorization request.
const CodeChallengeMethodS256 = "S256"

// The error codes an OAuth 2.0 error response carries in its error parameter, as RFC 6749 §4.1.2.1
// and §5.2 define them.
const (
	// OAuthErrorInvalidRequest refuses a request that misses a required parameter, repeats one, or
	// gives one a value the server does not support.
	OAuthErrorInvalidRequest = "invalid_request"
	// OAuthErrorInvalidClient refuses a token request whose client does not authenticate.
	OAuthErrorInvalidClient = "invalid_client"
	// OAuthErrorInvalidGrant refuses a code or a refresh token that is invalid, expired, revoked,
	// issued to another client, or bound to another redirect URI.
	OAuthErrorInvalidGrant = "invalid_grant"
	// OAuthErrorUnsupportedGrantType refuses a token request for a grant other than
	// authorization_code and refresh_token.
	OAuthErrorUnsupportedGrantType = "unsupported_grant_type"
	// OAuthErrorUnsupportedResponseType refuses an authorization request for a response other than a
	// code.
	OAuthErrorUnsupportedResponseType = "unsupported_response_type"
	// OAuthErrorInvalidScope refuses a request that names a scope: the server grants none.
	OAuthErrorInvalidScope = "invalid_scope"
)

// An OAuthError refuses an OAuth 2.0 request with an error response of RFC 6749: an error code, and
// a description for the client's developer. Two OAuthErrors match under errors.Is when their code
// and description do, whatever their redirect URI.
type OAuthError struct {
	// Code is the error parameter, such as invalid_grant.
	Code string
	// Description is the error_description parameter: ASCII without quotes or backslashes, as RFC
	// 6749 §5.2 allows.
	Description string
	// RedirectURI is where the account platform sends an authorization error response. Empty, the
	// account platform must not redirect, and shows the error to the person instead
	// (RFC 6749 §4.1.2.1).
	RedirectURI string
}

func (err *OAuthError) Error() string {
	return err.Code + ": " + err.Description
}

func (err *OAuthError) Is(target error) bool {
	other, ok := target.(*OAuthError)

	return ok && other.Code == err.Code && other.Description == err.Description
}

// redirectTo returns err, sent to redirectURI.
func (err *OAuthError) redirectTo(redirectURI string) *OAuthError {
	return &OAuthError{Code: err.Code, Description: err.Description, RedirectURI: redirectURI}
}

// The refusals every OAuth operation shares.
var (
	// ErrOAuthRepeatedParameter is returned by the OAuth operations for a request that gives a
	// parameter more than once (RFC 6749 §3.1, §3.2).
	ErrOAuthRepeatedParameter = &OAuthError{Code: OAuthErrorInvalidRequest, Description: "a parameter is repeated"}
	// ErrOAuthMalformedRequest answers a request whose parameters cannot be read: a malformed query
	// string, or a token request whose body is not application/x-www-form-urlencoded (RFC 6749 §3.2).
	ErrOAuthMalformedRequest = &OAuthError{
		Code: OAuthErrorInvalidRequest, Description: "the parameters are not form-urlencoded",
	}
	// ErrOAuthScope is returned by the OAuth operations for a request that names a scope. The server
	// grants none, so an authorization request asks for more than it gets, and a refresh for more
	// than the original grant (RFC 6749 §3.3, §6).
	ErrOAuthScope = &OAuthError{Code: OAuthErrorInvalidScope, Description: "no scope is granted"}
)

// ErrAuthorizationRequestForbiddenCaller is returned by [AuthorizationRequestValidate.Exec] and
// [AuthorizationCodeCreate.Exec] when the caller is not the account platform. A server that
// verifies no service token sees no caller, and refuses too.
var ErrAuthorizationRequestForbiddenCaller = errors.New("caller is not the account platform")

// The refusals of an authorization request that the account platform shows the person, without
// redirecting anywhere (RFC 6749 §4.1.2.1). [AuthorizationRequestValidate.Exec] and
// [AuthorizationCodeCreate.Exec] return them.
var (
	// ErrAuthorizationRequestUnknownClient is returned for a client_id that is missing, repeated or
	// unregistered.
	ErrAuthorizationRequestUnknownClient = &OAuthError{Code: OAuthErrorInvalidRequest, Description: "unknown client"}
	// ErrAuthorizationRequestInvalidRedirectURI is returned for a redirect_uri that is repeated, or
	// other than the client's one as an exact string (RFC 9700 §4.1.3).
	ErrAuthorizationRequestInvalidRedirectURI = &OAuthError{
		Code: OAuthErrorInvalidRequest, Description: "redirect uri is not the client's",
	}
)

// The refusals of an authorization request that the account platform redirects to the client
// (RFC 6749 §4.1.2.1), as the RedirectURI of the returned [OAuthError] names.
// [AuthorizationRequestValidate.Exec] and [AuthorizationCodeCreate.Exec] return them, and
// [ErrOAuthRepeatedParameter] and [ErrOAuthScope] too.
var (
	// ErrAuthorizationRequestNoResponseType is returned for a request without response_type.
	ErrAuthorizationRequestNoResponseType = &OAuthError{
		Code: OAuthErrorInvalidRequest, Description: "response_type is required",
	}
	// ErrAuthorizationRequestUnsupportedResponseType is returned for a response_type other than code.
	ErrAuthorizationRequestUnsupportedResponseType = &OAuthError{
		Code: OAuthErrorUnsupportedResponseType, Description: "response_type must be code",
	}
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

// checkAccountPlatform refuses any caller but the account platform, and no caller.
func checkAccountPlatform(oauth authconfig.OAuth, caller string) error {
	if caller == "" || caller != oauth.AccountPlatformCaller {
		return fmt.Errorf("%w: %q", ErrAuthorizationRequestForbiddenCaller, caller)
	}

	return nil
}

// setClientAttribute names on span the client of the authorization request params carry, once the
// registry holds it. An unregistered client_id is the caller's own value, and stays out of the
// attributes the Services tab groups by.
func setClientAttribute(span trace.Span, clients map[string]authconfig.Client, params url.Values) {
	clientID, ok := oauthParam(params, "client_id")
	if _, registered := clients[clientID]; ok && registered {
		span.SetAttributes(attribute.String("client.id", clientID))
	}
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
		return ErrOAuthScope
	case authorization.CodeChallenge == "":
		return ErrAuthorizationRequestNoCodeChallenge
	case authorization.CodeChallengeMethod != CodeChallengeMethodS256:
		return ErrAuthorizationRequestUnsupportedTransform
	case !s256Challenge.MatchString(authorization.CodeChallenge):
		return ErrAuthorizationRequestInvalidCodeChallenge
	}

	return nil
}

// oauthParam returns the value of the parameter name in params. A parameter given without a value
// reads as omitted (RFC 6749 §3.1). It reports false for a parameter given more than once.
func oauthParam(params url.Values, name string) (string, bool) {
	values := params[name]
	if len(values) > 1 {
		return "", false
	}

	if len(values) == 0 {
		return "", true
	}

	return values[0], true
}

// oauthParamsRepeated reports whether params repeats one of names. It ignores the other parameters:
// the server ignores a parameter it does not know (RFC 6749 §3.1, §3.2).
func oauthParamsRepeated(params url.Values, names ...string) bool {
	for _, name := range names {
		if len(params[name]) > 1 {
			return true
		}
	}

	return false
}
