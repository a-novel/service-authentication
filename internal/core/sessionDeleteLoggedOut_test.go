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

func TestSessionDeleteLoggedOut(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	type daoMock struct {
		err error
	}

	testCases := []struct {
		name string

		daoMock *daoMock

		expectErr error
	}{
		{
			name: "Success",

			daoMock: &daoMock{},
		},
		{
			name: "Error/Dao",

			daoMock: &daoMock{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockSessionDeleteLoggedOutDao(t)

			mockDao.EXPECT().
				Exec(mock.Anything, &dao.SessionDeleteLoggedOutRequest{UserID: userID}).
				Return(testCase.daoMock.err)

			service := core.NewSessionDeleteLoggedOut(mockDao)

			err := service.Exec(t.Context(), &core.SessionDeleteLoggedOutRequest{UserID: userID})
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
