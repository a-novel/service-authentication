package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/v2/internal/services/mocks"
)

func TestCredentialsGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp *dao.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *services.CredentialsGetRequest

		repositoryMock *repositoryMock

		expect    *services.Credentials
		expectErr error
	}{
		{
			name: "Success",

			request: &services.CredentialsGetRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			repositoryMock: &repositoryMock{
				resp: &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "user1@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &services.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "user1@email.com",
				Role:      config.RoleUser,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error",

			request: &services.CredentialsGetRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

			ctx := t.Context()

			repository := servicesmocks.NewMockCredentialsGetRepository(t)

			if testCase.repositoryMock != nil {
				repository.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: testCase.request.ID}).
					Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
			}

			service := services.NewCredentialsGet(repository)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repository.AssertExpectations(t)
		})
	}
}
