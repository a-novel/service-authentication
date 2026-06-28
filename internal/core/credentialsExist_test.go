package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestCredentialsExist(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp bool
		err  error
	}

	testCases := []struct {
		name string

		request *core.CredentialsExistRequest

		daoMock *daoMock

		expect    bool
		expectErr error
	}{
		{
			name: "Success",

			request: &core.CredentialsExistRequest{
				Email: "user@provider.com",
			},

			daoMock: &daoMock{
				resp: true,
			},

			expect: true,
		},
		{
			name: "Error",

			request: &core.CredentialsExistRequest{
				Email: "user@provider.com",
			},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockCredentialsExistDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsExistRequest{
						Email: testCase.request.Email,
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewCredentialsExist(mockDao)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
		})
	}
}
