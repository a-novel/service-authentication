package core_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestSessionDelete(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	sessionID := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")

	type daoSelectMock struct {
		resp *dao.Session
		err  error
	}

	type daoMock struct {
		err error
	}

	testCases := []struct {
		name string

		daoSelectMock *daoSelectMock
		daoMock       *daoMock

		expectErr error
	}{
		{
			name: "Success",

			daoSelectMock: &daoSelectMock{resp: &dao.Session{ID: sessionID, UserID: userID, LoggedOut: true}},
			daoMock:       &daoMock{},
		},
		{
			// An active session stays: removing it would hide a live session from its account.
			name: "Error/Active",

			daoSelectMock: &daoSelectMock{resp: &dao.Session{ID: sessionID, UserID: userID}},

			expectErr: core.ErrSessionActive,
		},
		{
			// The lookup matches the caller too, so another account's session answers as an unknown one.
			name: "Error/NotFound",

			daoSelectMock: &daoSelectMock{err: dao.ErrSessionSelectNotFound},

			expectErr: core.ErrSessionNotFound,
		},
		{
			// A concurrent request removed the row first.
			name: "Error/AlreadyDeleted",

			daoSelectMock: &daoSelectMock{resp: &dao.Session{ID: sessionID, UserID: userID, LoggedOut: true}},
			daoMock:       &daoMock{err: dao.ErrSessionDeleteNotFound},

			expectErr: core.ErrSessionNotFound,
		},
		{
			name: "Error/Delete",

			daoSelectMock: &daoSelectMock{resp: &dao.Session{ID: sessionID, UserID: userID, LoggedOut: true}},
			daoMock:       &daoMock{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockSessionDeleteDao(t)
			mockDaoSelect := coremocks.NewMockSessionDeleteDaoSelect(t)

			mockDaoSelect.EXPECT().
				Exec(mock.Anything, &dao.SessionSelectRequest{ID: sessionID, UserID: userID}).
				Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.SessionDeleteRequest{ID: sessionID, UserID: userID}).
					Return(testCase.daoMock.err)
			}

			service := core.NewSessionDelete(mockDao, mockDaoSelect)

			err := service.Exec(t.Context(), &core.SessionDeleteRequest{UserID: userID, ID: sessionID})
			require.ErrorIs(t, err, testCase.expectErr)

			mockDao.AssertExpectations(t)
			mockDaoSelect.AssertExpectations(t)
		})
	}
}
