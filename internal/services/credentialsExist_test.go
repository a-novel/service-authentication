package services_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/v2/internal/services/mocks"
)

func TestCredentialsExist(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp bool
		err  error
	}

	testCases := []struct {
		name string

		request *services.CredentialsExistRequest

		repositoryMock *repositoryMock

		expect    bool
		expectErr error
	}{
		{
			name: "Success",

			request: &services.CredentialsExistRequest{
				Email: "user@provider.com",
			},

			repositoryMock: &repositoryMock{
				resp: true,
			},

			expect: true,
		},
		{
			name: "Error",

			request: &services.CredentialsExistRequest{
				Email: "user@provider.com",
			},

			repositoryMock: &repositoryMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repository := servicesmocks.NewMockCredentialsExistRepository(t)

			if testCase.repositoryMock != nil {
				repository.EXPECT().
					Exec(mock.Anything, &dao.CredentialsExistRequest{
						Email: testCase.request.Email,
					}).
					Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
			}

			service := services.NewCredentialsExist(repository)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repository.AssertExpectations(t)
		})
	}
}
