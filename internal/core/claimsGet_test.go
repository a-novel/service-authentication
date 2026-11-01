package core_test

import (
	"errors"
	"testing"

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

	type daoMock struct {
		jtis []string

		resp bool
		err  error
	}

	testCases := []struct {
		name string

		request *core.ClaimsGetRequest

		daoMock *daoMock

		expect    *session.Claims
		expectErr error
		// expectNotErr is a sentinel the returned error must not carry.
		expectNotErr error
	}{
		{
			name: "Success",

			request: &core.ClaimsGetRequest{Claims: ownSessionClaims},

			daoMock: &daoMock{jtis: []string{"refresh-token-id", "refresh-token-id"}},

			expect: ownSessionClaims,
		},
		{
			name: "Success/Session",

			request: &core.ClaimsGetRequest{Claims: sessionClaims},

			daoMock: &daoMock{jtis: []string{"refresh-token-id", "session-id"}},

			expect: sessionClaims,
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

			service := core.NewClaimsGet(mockDao)

			claims, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, claims)

			if testCase.expectNotErr != nil {
				require.NotErrorIs(t, err, testCase.expectNotErr)
			}

			mockDao.AssertExpectations(t)
		})
	}
}
