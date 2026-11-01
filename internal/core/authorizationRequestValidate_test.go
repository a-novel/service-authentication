package core_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
)

func TestAuthorizationRequestValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *core.AuthorizationRequestValidateRequest

		expectErr error
	}{
		{
			name: "Success",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: accountPlatformCaller, ClientID: "studio", RedirectURI: studioRedirectURI,
			},
		},
		{
			// A server that verifies no service token sees no caller.
			name: "Error/NoCaller",

			request: &core.AuthorizationRequestValidateRequest{ClientID: "studio", RedirectURI: studioRedirectURI},

			expectErr: core.ErrAuthorizationRequestForbiddenCaller,
		},
		{
			name: "Error/ClientCaller",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: studioCaller, ClientID: "studio", RedirectURI: studioRedirectURI,
			},

			expectErr: core.ErrAuthorizationRequestForbiddenCaller,
		},
		{
			name: "Error/UnknownClient",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: accountPlatformCaller, ClientID: "blog", RedirectURI: studioRedirectURI,
			},

			expectErr: core.ErrAuthorizationRequestUnknownClient,
		},
		{
			name: "Error/RedirectURIPrefix",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: accountPlatformCaller, ClientID: "studio", RedirectURI: "https://studio.example.com/auth",
			},

			expectErr: core.ErrAuthorizationRequestRedirectURIMismatch,
		},
		{
			name: "Error/RedirectURIExtraPath",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: accountPlatformCaller, ClientID: "studio", RedirectURI: studioRedirectURI + "/next",
			},

			expectErr: core.ErrAuthorizationRequestRedirectURIMismatch,
		},
		{
			name: "Error/RedirectURITrailingSlash",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: accountPlatformCaller, ClientID: "studio", RedirectURI: studioRedirectURI + "/",
			},

			expectErr: core.ErrAuthorizationRequestRedirectURIMismatch,
		},
		{
			name: "Error/RedirectURIQuery",

			request: &core.AuthorizationRequestValidateRequest{
				Caller:      accountPlatformCaller,
				ClientID:    "studio",
				RedirectURI: studioRedirectURI + "?next=https://evil.example.com",
			},

			expectErr: core.ErrAuthorizationRequestRedirectURIMismatch,
		},
		{
			name: "Error/RedirectURIOtherHost",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: accountPlatformCaller, ClientID: "studio", RedirectURI: "https://evil.example.com/auth/callback",
			},

			expectErr: core.ErrAuthorizationRequestRedirectURIMismatch,
		},
		{
			// The registered host as user information sends the browser to another host.
			name: "Error/RedirectURIUserInfo",

			request: &core.AuthorizationRequestValidateRequest{
				Caller:      accountPlatformCaller,
				ClientID:    "studio",
				RedirectURI: "https://studio.example.com@evil.example.com/auth/callback",
			},

			expectErr: core.ErrAuthorizationRequestRedirectURIMismatch,
		},
		{
			name: "Error/RedirectURIScheme",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: accountPlatformCaller, ClientID: "studio", RedirectURI: "http://studio.example.com/auth/callback",
			},

			expectErr: core.ErrAuthorizationRequestRedirectURIMismatch,
		},
		{
			// A registered redirect URI belongs to its own client only.
			name: "Error/RedirectURIOtherClient",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: accountPlatformCaller, ClientID: "studio", RedirectURI: forumRedirectURI,
			},

			expectErr: core.ErrAuthorizationRequestRedirectURIMismatch,
		},
		{
			name: "Error/NoClientID",

			request: &core.AuthorizationRequestValidateRequest{
				Caller: accountPlatformCaller, RedirectURI: studioRedirectURI,
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/RedirectURITooLong",

			request: &core.AuthorizationRequestValidateRequest{
				Caller:      accountPlatformCaller,
				ClientID:    "studio",
				RedirectURI: studioRedirectURI + strings.Repeat("a", 2048),
			},

			expectErr: core.ErrInvalidRequest,
		},
	}

	service := core.NewAuthorizationRequestValidate(oauthPreset)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
