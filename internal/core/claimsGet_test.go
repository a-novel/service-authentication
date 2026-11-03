package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestClaimsGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001"))

	// ownSessionClaims has no session claim: the session its refresh token opened stands in for it.
	ownSessionClaims := &session.Claims{UserID: userID, Roles: []string{"user"}, RefreshTokenID: "refresh-token-id"}
	sessionClaims := &session.Claims{
		UserID:         userID,
		Roles:          []string{"user"},
		RefreshTokenID: "refresh-token-id",
		SessionID:      "session-id",
	}
	anonymousClaims := &session.Claims{Roles: []string{"anon"}}
	liveSessionClaims := changed(*sessionClaims, func(claims *session.Claims) {
		claims.SessionExp = time.Now().Add(time.Hour).Unix()
	})
	endedSessionClaims := changed(*sessionClaims, func(claims *session.Claims) {
		claims.SessionExp = time.Now().Add(-time.Second).Unix()
	})

	type daoMock struct {
		jtis []string

		resp bool
		err  error
	}

	type credentialsMock struct {
		resp *dao.Credentials
		err  error
	}

	// The account's password and email last changed before every token of the test was issued.
	credentials := &credentialsMock{resp: &dao.Credentials{ID: *userID}}

	testCases := []struct {
		name string

		request *core.ClaimsGetRequest

		daoMock         *daoMock
		credentialsMock *credentialsMock

		expect    *session.Claims
		expectErr error
		// expectNotErr is a sentinel the returned error must not carry.
		expectNotErr error
	}{
		{
			name: "Success",

			request: &core.ClaimsGetRequest{Claims: ownSessionClaims},

			daoMock:         &daoMock{jtis: []string{"refresh-token-id", "refresh-token-id"}},
			credentialsMock: credentials,

			expect: ownSessionClaims,
		},
		{
			name: "Success/Session",

			request: &core.ClaimsGetRequest{Claims: sessionClaims},

			daoMock:         &daoMock{jtis: []string{"refresh-token-id", "session-id"}},
			credentialsMock: credentials,

			expect: sessionClaims,
		},
		{
			name: "Success/SessionExp",

			request: &core.ClaimsGetRequest{Claims: liveSessionClaims},

			daoMock:         &daoMock{jtis: []string{"refresh-token-id", "session-id"}},
			credentialsMock: credentials,

			expect: liveSessionClaims,
		},
		{
			// The access token outlives its session by up to its lifetime, and the purge may already
			// have dropped the session's revocation: its end refuses it on its own.
			name: "Error/SessionEnded",

			request: &core.ClaimsGetRequest{Claims: endedSessionClaims},

			expectErr: core.ErrClaimsGetRevoked,
		},
		{
			// An anonymous token belongs to no session, so nothing revokes it.
			name: "Success/Anonymous",

			request: &core.ClaimsGetRequest{Claims: anonymousClaims},

			expect: anonymousClaims,
		},
		{
			// Signing out from any token of the session refuses this one on its next check.
			name: "Error/Revoked",

			request: &core.ClaimsGetRequest{Claims: sessionClaims},

			daoMock: &daoMock{jtis: []string{"refresh-token-id", "session-id"}, resp: true},

			expectErr: core.ErrClaimsGetRevoked,
		},
		{
			name: "Error/Dao",

			request: &core.ClaimsGetRequest{Claims: ownSessionClaims},

			daoMock: &daoMock{jtis: []string{"refresh-token-id", "refresh-token-id"}, err: errFoo},

			expectErr:    errFoo,
			expectNotErr: core.ErrClaimsGetRevoked,
		},
		{
			// A password or email change after the token was issued ended its session.
			name: "Error/CredentialsChanged",

			request: &core.ClaimsGetRequest{Claims: sessionClaims},

			daoMock: &daoMock{jtis: []string{"refresh-token-id", "session-id"}},
			credentialsMock: &credentialsMock{
				resp: &dao.Credentials{ID: *userID, CredentialsUpdatedAt: time.Now()},
			},

			expectErr: core.ErrClaimsGetRevoked,
		},
		{
			name: "Error/AccountDeleted",

			request: &core.ClaimsGetRequest{Claims: sessionClaims},

			daoMock:         &daoMock{jtis: []string{"refresh-token-id", "session-id"}},
			credentialsMock: &credentialsMock{err: dao.ErrCredentialsSelectNotFound},

			expectErr: core.ErrClaimsGetRevoked,
		},
		{
			name: "Error/CredentialsSelect",

			request: &core.ClaimsGetRequest{Claims: sessionClaims},

			daoMock:         &daoMock{jtis: []string{"refresh-token-id", "session-id"}},
			credentialsMock: &credentialsMock{err: errFoo},

			expectErr:    errFoo,
			expectNotErr: core.ErrClaimsGetRevoked,
		},
		{
			name: "Error/NoClaims",

			request: &core.ClaimsGetRequest{},

			expectErr: core.ErrInvalidRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockClaimsGetDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.RefreshTokenRevocationExistRequest{JTIs: testCase.daoMock.jtis}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			mockDaoCredentials := coremocks.NewMockClaimsGetDaoCredentials(t)

			if testCase.credentialsMock != nil {
				mockDaoCredentials.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: *userID}).
					Return(testCase.credentialsMock.resp, testCase.credentialsMock.err)
			}

			service := core.NewClaimsGet(mockDao, mockDaoCredentials)

			claims, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, claims)

			if testCase.expectNotErr != nil {
				require.NotErrorIs(t, err, testCase.expectNotErr)
			}

			mockDao.AssertExpectations(t)
			mockDaoCredentials.AssertExpectations(t)
		})
	}
}
