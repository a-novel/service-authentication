package core_test

import (
	"crypto/sha256"
	"errors"
	"strings"
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
	// The S256 challenge of the verifier in RFC 7636's appendix B.
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	validRequest := core.AuthorizationCodeCreateRequest{
		Caller:              accountPlatformCaller,
		UserID:              userID,
		SessionID:           "session-id",
		SessionExp:          sessionExp,
		ClientID:            "studio",
		RedirectURI:         studioRedirectURI,
		CodeChallenge:       challenge,
		CodeChallengeMethod: core.CodeChallengeMethodS256,
	}

	type daoMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.AuthorizationCodeCreateRequest

		daoMock *daoMock

		expectErr error
	}{
		{
			name: "Success",

			request: &validRequest,

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

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) { request.Caller = studioCaller }),

			expectErr: core.ErrAuthorizationRequestForbiddenCaller,
		},
		{
			name: "Error/UnknownClient",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) { request.ClientID = "blog" }),

			expectErr: core.ErrAuthorizationRequestUnknownClient,
		},
		{
			name: "Error/RedirectURIMismatch",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.RedirectURI = studioRedirectURI + "?next=/"
			}),

			expectErr: core.ErrAuthorizationRequestRedirectURIMismatch,
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

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoSession",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) { request.SessionID = "" }),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/PlainMethod",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.CodeChallengeMethod = "plain"
			}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoMethod",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.CodeChallengeMethod = ""
			}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ChallengeTooShort",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.CodeChallenge = challenge[:42]
			}),

			expectErr: core.ErrInvalidRequest,
		},
		{
			// A padded or standard base64 value is no S256 challenge.
			name: "Error/ChallengeAlphabet",

			request: changed(validRequest, func(request *core.AuthorizationCodeCreateRequest) {
				request.CodeChallenge = strings.Repeat("+", 43)
			}),

			expectErr: core.ErrInvalidRequest,
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
			// person, the session, the client, its redirect URI and the challenge.
			require.Len(t, res.Code, 43)

			id := sha256.Sum256([]byte(res.Code))

			require.Equal(t, id[:], inserted.ID)
			require.Equal(t, userID, inserted.UserID)
			require.Equal(t, "session-id", inserted.SessionID)
			require.True(t, time.Unix(sessionExp, 0).Equal(inserted.SessionExpiresAt))
			require.Equal(t, "studio", inserted.ClientID)
			require.Equal(t, studioRedirectURI, inserted.RedirectURI)
			require.Equal(t, challenge, inserted.CodeChallenge)
			require.WithinDuration(t, time.Now().Add(core.AuthorizationCodeTTL), inserted.ExpiresAt, 5*time.Second)
		})
	}
}
