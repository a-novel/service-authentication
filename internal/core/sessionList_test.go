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

func TestSessionList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	now := time.Now().Round(time.Second)
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	currentID := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	otherID := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	entities := []*dao.Session{
		{
			ID: otherID, UserID: userID, UserAgent: "Chrome", CreatedAt: now, LastActiveAt: now, ExpiresAt: now,
			LoggedOut: true,
		},
		{
			ID: currentID, UserID: userID, UserAgent: "Firefox", IPAddress: "192.0.2.1", Region: "FR", City: "Paris",
			CreatedAt: now, LastActiveAt: now, ExpiresAt: now,
		},
	}

	type daoMock struct {
		resp []*dao.Session
		err  error
	}

	testCases := []struct {
		name string

		request *core.SessionListRequest

		daoMock *daoMock

		expect    []*core.Session
		expectErr error
	}{
		{
			// The list reads the caller's own sessions, marks the one their token belongs to, and keeps the
			// logged-out ones.
			name: "Success",

			request: &core.SessionListRequest{UserID: userID, SessionID: currentID.String()},

			daoMock: &daoMock{resp: entities},

			expect: []*core.Session{
				{ID: otherID, LoggedOut: true, UserAgent: "Chrome", CreatedAt: now, LastActiveAt: now, ExpiresAt: now},
				{
					ID: currentID, Current: true, UserAgent: "Firefox", IPAddress: "192.0.2.1", Region: "FR", City: "Paris",
					CreatedAt: now, LastActiveAt: now, ExpiresAt: now,
				},
			},
		},
		{
			name: "Success/Empty",

			request: &core.SessionListRequest{UserID: userID, SessionID: currentID.String()},

			daoMock: &daoMock{resp: []*dao.Session{}},

			expect: []*core.Session{},
		},
		{
			name: "Error/Dao",

			request: &core.SessionListRequest{UserID: userID, SessionID: currentID.String()},

			daoMock: &daoMock{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockSessionListDao(t)

			mockDao.EXPECT().
				Exec(mock.Anything, &dao.SessionListRequest{UserID: userID, Limit: core.SessionListLimit}).
				Return(testCase.daoMock.resp, testCase.daoMock.err)

			service := core.NewSessionList(mockDao)

			res, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			mockDao.AssertExpectations(t)
		})
	}
}
