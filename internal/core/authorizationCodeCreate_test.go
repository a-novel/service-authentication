package core_test

import (
	"crypto/sha256"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestAuthorizationCodeCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sessionExp := time.Now().Add(time.Hour).Unix()

	validRequest := core.AuthorizationCodeCreateRequest{
		Caller:     accountPlatformCaller,
		UserID:     userID,
		SessionID:  "session-id",
		SessionExp: sessionExp,
		Params:     authorizationParams(nil),
	}

	type daoMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.AuthorizationCodeCreateRequest

		daoMock *daoMock

		expectErr error
		// expectRedirectURIInRequest is the stored record of whether the request named its redirect URI.
		expectRedirectURIInRequest bool
	}{
		{
			name: "Success",

			request: &validRequest,

			daoMock: &daoMock{},

			expectRedirectURIInRequest: true,
		},
		{
			// The code goes to the client's one redirect URI, and the token request may then omit it.
			name: "Success/NoRedirectURI",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.Params = authorizationParams(func(params url.Values) { params.Del("redirect_uri") })
			}),

			daoMock: &daoMock{},
		},
		{
			name: "Error/Dao",

			request: &validRequest,

			daoMock: &daoMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			// A server that verifies no service token sees no caller.
			name: "Error/NoCaller",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) { request.Caller = "" }),

			expectErr: core.ErrAuthorizationRequestForbiddenCaller,
		},
		{
			name: "Error/ClientCaller",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.Caller = studioCaller
			}),

			expectErr: core.ErrAuthorizationRequestForbiddenCaller,
		},
		{
			// The request is checked again, as [core.AuthorizationRequestValidate] checks it.
			name: "Error/RedirectURIMismatch",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.Params = withRedirectURI(studioRedirectURI + "?next=/")
			}),

			expectErr: core.ErrAuthorizationRequestInvalidRedirectURI,
		},
		{
			name: "Error/PlainMethod",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.Params = authorizationParams(func(params url.Values) { params.Set("code_challenge_method", "plain") })
			}),

			expectErr: core.ErrAuthorizationRequestUnsupportedTransform,
		},
		{
			// The access token is still valid, but its session's refresh token has expired.
			name: "Error/SessionEnded",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.SessionExp = time.Now().Add(-time.Minute).Unix()
			}),

			expectErr: core.ErrAuthorizationCodeCreateSessionEnded,
		},
		{
			// An access token signed before sessions carried their end.
			name: "Error/NoSessionExp",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) { request.SessionExp = 0 }),

			expectErr: core.ErrAuthorizationCodeCreateSessionEnded,
		},
		{
			name: "Error/NoSession",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) { request.SessionID = "" }),

			expectErr: core.ErrAuthorizationCodeCreateSessionEnded,
		},
		{
			name: "Error/NoUser",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) { request.UserID = uuid.Nil }),

			expectErr: core.ErrAuthorizationCodeCreateSessionEnded,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockAuthorizationCodeCreateDao(t)

			var inserted *dao.AuthorizationCodeInsertRequest

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.AuthorizationCodeInsertRequest) bool {
						inserted = request

						return true
					})).
					Return(testCase.daoMock.err)
			}

			service := core.NewAuthorizationCodeCreate(mockDao, oauthPreset)

			res, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			mockDao.AssertExpectations(t)

			if testCase.expectErr != nil {
				require.Nil(t, res)

				return
			}

			// The code is 256 random bits, and only its SHA-256 reaches the database, bound to the
			// person, the session, the client, its redirect URI and the PKCE challenge and method.
			require.Len(t, res.Code, 43)
			require.Equal(t, studioRedirectURI, res.RedirectURI)

			id := sha256.Sum256([]byte(res.Code))

			require.Equal(t, id[:], inserted.ID)
			require.Equal(t, userID, inserted.UserID)
			require.Equal(t, "session-id", inserted.SessionID)
			require.True(t, time.Unix(sessionExp, 0).Equal(inserted.SessionExpiresAt))
			require.Equal(t, "studio", inserted.ClientID)
			require.Equal(t, studioRedirectURI, inserted.RedirectURI)
			require.Equal(t, testCase.expectRedirectURIInRequest, inserted.RedirectURIInRequest)
			require.Equal(t, rfc7636Challenge, inserted.CodeChallenge)
			require.Equal(t, core.CodeChallengeMethodS256, inserted.CodeChallengeMethod)
			require.WithinDuration(t, time.Now().Add(core.AuthorizationCodeTTL), inserted.ExpiresAt, 5*time.Second)
		})
	}
}
