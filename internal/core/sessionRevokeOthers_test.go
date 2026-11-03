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

func TestSessionRevokeOthers(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	leeway := 5 * time.Minute
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	currentID := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")

	type daoMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.SessionRevokeOthersRequest

		daoMock *daoMock

		expectErr bool
	}{
		{
			// The caller's own session stays live.
			name: "Success",

			request: &core.SessionRevokeOthersRequest{UserID: userID, SessionID: currentID.String()},

			daoMock: &daoMock{},
		},
		{
			// Every session ID is a UUID, so another value is a fault, and revokes nothing.
			name: "Error/MalformedSessionID",

			request: &core.SessionRevokeOthersRequest{UserID: userID, SessionID: "session"},

			expectErr: true,
		},
		{
			name: "Error/Dao",

			request: &core.SessionRevokeOthersRequest{UserID: userID, SessionID: currentID.String()},

			daoMock: &daoMock{err: errFoo},

			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockSessionRevokeOthersDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *dao.SessionRevokeOthersRequest) bool {
						return request.UserID == userID && request.KeepID == currentID && request.Leeway == leeway &&
							time.Since(request.Now) < time.Minute
					})).
					Return(testCase.daoMock.err)
			}

			service := core.NewSessionRevokeOthers(mockDao, leeway)

			err := service.Exec(t.Context(), testCase.request)
			require.Equal(t, testCase.expectErr, err != nil, err)

			if testCase.daoMock != nil && testCase.daoMock.err != nil {
				require.ErrorIs(t, err, testCase.daoMock.err)
			}

			mockDao.AssertExpectations(t)
		})
	}
}
