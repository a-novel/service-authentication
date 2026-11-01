package core_test

import (
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

func TestAuthorizationRequestValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		caller string
		params url.Values

		expect    *core.AuthorizationRequest
		expectErr error
		// expectRedirectURI is where the refusal redirects, empty for one the person sees.
		expectRedirectURI string
	}{
		{
			name: "Success",

			caller: accountPlatformCaller,
			params: authorizationParams(nil),

			expect: &core.AuthorizationRequest{
				ClientID:             "studio",
				RedirectURI:          studioRedirectURI,
				RedirectURIInRequest: true,
				CodeChallenge:        rfc7636Challenge,
				CodeChallengeMethod:  core.CodeChallengeMethodS256,
			},
		},
		{
			// A client with one registered redirect URI may omit it (RFC 6749 §3.1.2.3).
			name: "Success/NoRedirectURI",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Del("redirect_uri") }),

			expect: &core.AuthorizationRequest{
				ClientID:            "studio",
				RedirectURI:         studioRedirectURI,
				CodeChallenge:       rfc7636Challenge,
				CodeChallengeMethod: core.CodeChallengeMethodS256,
			},
		},
		{
			// A parameter without a value reads as omitted, and an unknown one is ignored, repeated or
			// not (RFC 6749 §3.1).
			name: "Success/EmptyAndUnknownParameters",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) {
				params.Set("scope", "")
				params["prompt"] = []string{"login", "consent"}
			}),

			expect: &core.AuthorizationRequest{
				ClientID:             "studio",
				RedirectURI:          studioRedirectURI,
				RedirectURIInRequest: true,
				CodeChallenge:        rfc7636Challenge,
				CodeChallengeMethod:  core.CodeChallengeMethodS256,
			},
		},
		{
			// A server that verifies no service token sees no caller.
			name: "Error/NoCaller",

			params: authorizationParams(nil),

			expectErr: core.ErrAuthorizationRequestForbiddenCaller,
		},
		{
			name: "Error/ClientCaller",

			caller: studioCaller,
			params: authorizationParams(nil),

			expectErr: core.ErrAuthorizationRequestForbiddenCaller,
		},
		{
			name: "Error/NoClientID",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Del("client_id") }),

			expectErr: core.ErrAuthorizationRequestUnknownClient,
		},
		{
			name: "Error/UnknownClient",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Set("client_id", "blog") }),

			expectErr: core.ErrAuthorizationRequestUnknownClient,
		},
		{
			name: "Error/RepeatedClientID",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Add("client_id", "forum") }),

			expectErr: core.ErrAuthorizationRequestUnknownClient,
		},
		{
			name: "Error/RedirectURIPrefix",

			caller: accountPlatformCaller,
			params: withRedirectURI("https://studio.example.com/auth"),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			name: "Error/RedirectURIExtraPath",

			caller: accountPlatformCaller,
			params: withRedirectURI(studioRedirectURI + "/next"),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			name: "Error/RedirectURITrailingSlash",

			caller: accountPlatformCaller,
			params: withRedirectURI(studioRedirectURI + "/"),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			name: "Error/RedirectURIQuery",

			caller: accountPlatformCaller,
			params: withRedirectURI(studioRedirectURI + "?next=https://evil.example.com"),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			name: "Error/RedirectURIOtherHost",

			caller: accountPlatformCaller,
			params: withRedirectURI("https://evil.example.com/auth/callback"),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			// The registered host as user information sends the browser to another host.
			name: "Error/RedirectURIUserInfo",

			caller: accountPlatformCaller,
			params: withRedirectURI("https://studio.example.com@evil.example.com/auth/callback"),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			name: "Error/RedirectURIScheme",

			caller: accountPlatformCaller,
			params: withRedirectURI("http://studio.example.com/auth/callback"),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			name: "Error/RedirectURICase",

			caller: accountPlatformCaller,
			params: withRedirectURI("https://Studio.example.com/auth/callback"),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			// A registered redirect URI belongs to its own client only.
			name: "Error/RedirectURIOtherClient",

			caller: accountPlatformCaller,
			params: withRedirectURI(forumRedirectURI),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			name: "Error/RepeatedRedirectURI",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Add("redirect_uri", studioRedirectURI) }),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			name: "Error/RepeatedState",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Add("state", "other") }),

			expectErr:         core.ErrOAuthRepeatedParameter,
			expectRedirectURI: studioRedirectURI,
		},
		{
			name: "Error/NoResponseType",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Del("response_type") }),

			expectErr:         core.ErrAuthorizationRequestNoResponseType,
			expectRedirectURI: studioRedirectURI,
		},
		{
			name: "Error/TokenResponseType",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Set("response_type", "token") }),

			expectErr:         core.ErrAuthorizationRequestUnsupportedResponseType,
			expectRedirectURI: studioRedirectURI,
		},
		{
			name: "Error/Scope",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Set("scope", "openid") }),

			expectErr:         core.ErrOAuthScope,
			expectRedirectURI: studioRedirectURI,
		},
		{
			name: "Error/NoCodeChallenge",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Del("code_challenge") }),

			expectErr:         core.ErrAuthorizationRequestNoCodeChallenge,
			expectRedirectURI: studioRedirectURI,
		},
		{
			// Without a method, a request asks for plain (RFC 7636 §4.3).
			name: "Error/NoCodeChallengeMethod",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Del("code_challenge_method") }),

			expectErr:         core.ErrAuthorizationRequestUnsupportedTransform,
			expectRedirectURI: studioRedirectURI,
		},
		{
			name: "Error/PlainMethod",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Set("code_challenge_method", "plain") }),

			expectErr:         core.ErrAuthorizationRequestUnsupportedTransform,
			expectRedirectURI: studioRedirectURI,
		},
		{
			name: "Error/ChallengeTooShort",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) { params.Set("code_challenge", rfc7636Challenge[:42]) }),

			expectErr:         core.ErrAuthorizationRequestInvalidCodeChallenge,
			expectRedirectURI: studioRedirectURI,
		},
		{
			// Standard base64, padded or not, is no S256 challenge.
			name: "Error/ChallengeAlphabet",

			caller: accountPlatformCaller,
			params: authorizationParams(func(params url.Values) {
				params.Set("code_challenge", "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw+cM")
			}),

			expectErr:         core.ErrAuthorizationRequestInvalidCodeChallenge,
			expectRedirectURI: studioRedirectURI,
		},
	}

	service := core.NewAuthorizationRequestValidate(oauthPreset)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := &core.AuthorizationRequestValidateRequest{Caller: testCase.caller, Params: testCase.params}

			res, err := service.Exec(t.Context(), request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			var refusal *core.OAuthError
			if errors.As(err, &refusal) {
				require.Equal(t, testCase.expectRedirectURI, refusal.RedirectURI)
			}
		})
	}
}

// withRedirectURI returns the parameters of a valid authorization request, naming redirectURI.
func withRedirectURI(redirectURI string) url.Values {
	return authorizationParams(func(params url.Values) { params.Set("redirect_uri", redirectURI) })
}
