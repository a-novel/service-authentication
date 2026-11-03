package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestSessionRevoke(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	leeway := 5 * time.Minute
	expiresAt := time.Now().Add(24 * time.Hour).Round(time.Second)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")

	type daoMock struct {
		resp *dao.Session
		err  error
	}

	type revocationInsertMock struct {
		err error
	}

	testCases := []struct {
		name string

		daoMock              *daoMock
		revocationInsertMock *revocationInsertMock

		expectErr error
	}{
		{
			// The revocation lasts as long as the session's refresh tokens may.
			name: "Success",

			daoMock: &daoMock{resp: &dao.Session{ID: sessionID, UserID: userID, ExpiresAt: expiresAt}},

			revocationInsertMock: &revocationInsertMock{},
		},
		{
			// A session already signed out signs out again, without error.
			name: "Success/AlreadyRevoked",

			daoMock: &daoMock{
				resp: &dao.Session{ID: sessionID, UserID: userID, ExpiresAt: expiresAt, LoggedOut: true},
			},

			revocationInsertMock: &revocationInsertMock{},
		},
		{
			// An ended session answers as an unknown one.
			name: "Error/Ended",

			daoMock: &daoMock{
				resp: &dao.Session{ID: sessionID, UserID: userID, ExpiresAt: time.Now().Add(-time.Hour), LoggedOut: true},
			},

			expectErr: core.ErrSessionNotFound,
		},
		{
			// The lookup matches the caller too, so another account's session answers as an unknown one.
			name: "Error/NotFound",

			daoMock: &daoMock{err: dao.ErrSessionSelectNotFound},

			expectErr: core.ErrSessionNotFound,
		},
		{
			name: "Error/RevocationInsert",

			daoMock: &daoMock{resp: &dao.Session{ID: sessionID, UserID: userID, ExpiresAt: expiresAt}},

			revocationInsertMock: &revocationInsertMock{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockSessionRevokeDao(t)
			mockDaoRevocationInsert := coremocks.NewMockSessionRevokeDaoRevocationInsert(t)

			mockDao.EXPECT().
				Exec(mock.Anything, &dao.SessionSelectRequest{ID: sessionID, UserID: userID}).
				Return(testCase.daoMock.resp, testCase.daoMock.err)

			if testCase.revocationInsertMock != nil {
				mockDaoRevocationInsert.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.RefreshTokenRevocationInsertRequest) bool {
						return len(request.JTIs) == 1 && request.JTIs[0] == sessionID.String() &&
							request.ExpiresAt.Equal(expiresAt.Add(leeway)) && time.Since(request.Now) < time.Minute
					})).
					Return(testCase.revocationInsertMock.err)
			}

			service := core.NewSessionRevoke(mockDao, mockDaoRevocationInsert, leeway)

			err := service.Exec(t.Context(), &core.SessionRevokeRequest{UserID: userID, ID: sessionID})
			require.ErrorIs(t, err, testCase.expectErr)

			mockDao.AssertExpectations(t)
			mockDaoRevocationInsert.AssertExpectations(t)
		})
	}
}
