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
		Caller:         accountPlatformCaller,
		UserID:         userID,
		RefreshTokenID: "refresh-token-id",
		SessionID:      "session-id",
		SessionExp:     sessionExp,
		IssuedAt:       time.Now().Unix(),
		Params:         authorizationParams(nil),
	}

	// The account's password and email last changed before the access token was issued.
	credentials := &dao.Credentials{ID: userID, CredentialsUpdatedAt: time.Now().Add(-time.Hour)}

	type revocationMock struct {
		revoked bool
		err     error
	}

	type credentialsMock struct {
		resp *dao.Credentials
		err  error
	}

	type daoMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.AuthorizationCodeCreateRequest

		revocationMock  *revocationMock
		credentialsMock *credentialsMock
		daoMock         *daoMock

		expectErr error
		// expectRedirectURIInRequest is the stored record of whether the request named its redirect URI.
		expectRedirectURIInRequest bool
	}{
		{
			name: "Success",

			request: &validRequest,

			revocationMock:  &revocationMock{},
			credentialsMock: &credentialsMock{resp: credentials},
			daoMock:         &daoMock{},

			expectRedirectURIInRequest: true,
		},
		{
			// The code goes to the client's one redirect URI, and the token request may then omit it.
			name: "Success/NoRedirectURI",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.Params = authorizationParams(func(params url.Values) { params.Del("redirect_uri") })
			}),

			revocationMock:  &revocationMock{},
			credentialsMock: &credentialsMock{resp: credentials},
			daoMock:         &daoMock{},
		},
		{
			name: "Error/Dao",

			request: &validRequest,

			revocationMock:  &revocationMock{},
			credentialsMock: &credentialsMock{resp: credentials},
			daoMock:         &daoMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			// The password or email changed after the access token was issued: the session it belongs to
			// ended, so no client joins it.
			name: "Error/CredentialsChanged",

			request: &validRequest,

			revocationMock: &revocationMock{},
			credentialsMock: &credentialsMock{
				resp: &dao.Credentials{ID: userID, CredentialsUpdatedAt: time.Now().Add(time.Second)},
			},

			expectErr: core.ErrAuthorizationCodeCreateSessionEnded,
		},
		{
			name: "Error/AccountDeleted",

			request: &validRequest,

			revocationMock:  &revocationMock{},
			credentialsMock: &credentialsMock{err: dao.ErrCredentialsSelectNotFound},

			expectErr: core.ErrAuthorizationCodeCreateSessionEnded,
		},
		{
			name: "Error/CredentialsSelect",

			request: &validRequest,

			revocationMock:  &revocationMock{},
			credentialsMock: &credentialsMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			// The access token still verifies after a sign-out, but its session is revoked.
			name: "Error/SignedOut",

			request: &validRequest,

			revocationMock: &revocationMock{revoked: true},

			expectErr: core.ErrAuthorizationCodeCreateSessionEnded,
		},
		{
			name: "Error/RevocationExist",

			request: &validRequest,

			revocationMock: &revocationMock{err: errFoo},

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

			mockDaoRevocationExist := coremocks.NewMockAuthorizationCodeCreateDaoRevocationExist(t)
			mockDao := coremocks.NewMockAuthorizationCodeCreateDaoInsert(t)
			mockDaoCredentials := coremocks.NewMockAuthorizationCodeCreateDaoCredentials(t)

			if testCase.credentialsMock != nil {
				mockDaoCredentials.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: userID}).
					Return(testCase.credentialsMock.resp, testCase.credentialsMock.err)
			}

			if testCase.revocationMock != nil {
				revocationRequest := &dao.RefreshTokenRevocationExistRequest{JTIs: []string{"refresh-token-id", "session-id"}}

				mockDaoRevocationExist.EXPECT().
					Exec(mock.Anything, revocationRequest).
					Return(testCase.revocationMock.revoked, testCase.revocationMock.err)
			}

			var inserted *dao.AuthorizationCodeInsertRequest

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.AuthorizationCodeInsertRequest) bool {
						inserted = request

						return true
					})).
					Return(testCase.daoMock.err)
			}

			service := core.NewAuthorizationCodeCreate(mockDao, mockDaoRevocationExist, mockDaoCredentials, oauthPreset)

			res, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			mockDaoRevocationExist.AssertExpectations(t)
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
