package core

import (
	"net/url"
)

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

// ErrOAuthRepeatedParameter is returned by the OAuth operations for a request that gives a parameter
// more than once (RFC 6749 §3.1, §3.2).
var ErrOAuthRepeatedParameter = &OAuthError{Code: OAuthErrorInvalidRequest, Description: "a parameter is repeated"}

// ErrOAuthMalformedRequest answers a request whose parameters cannot be read: a malformed query
// string, or a token request whose body is not application/x-www-form-urlencoded (RFC 6749 §3.2).
var ErrOAuthMalformedRequest = &OAuthError{
	Code: OAuthErrorInvalidRequest, Description: "the parameters are not form-urlencoded",
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
