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

func TestCredentialsList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp []*dao.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *services.CredentialsListRequest

		repositoryMock *repositoryMock

		expect    []*services.Credentials
		expectErr error
	}{
		{
			name: "Success",

			request: &services.CredentialsListRequest{
				Limit:  10,
				Offset: 0,
				Roles:  []string{config.RoleUser},
			},

			repositoryMock: &repositoryMock{
				resp: []*dao.Credentials{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						Email:     "user3@email.com",
						Role:      config.RoleUser,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						Email:     "user2@email.com",
						Role:      config.RoleAdmin,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Email:     "user1@email.com",
						Role:      config.RoleUser,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expect: []*services.Credentials{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Email:     "user3@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user2@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "user1@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Error",

			request: &services.CredentialsListRequest{
				Limit:  10,
				Offset: 0,
				Roles:  []string{config.RoleUser},
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

			repository := servicesmocks.NewMockCredentialsListRepository(t)

			if testCase.repositoryMock != nil {
				repository.EXPECT().
					Exec(mock.Anything, &dao.CredentialsListRequest{
						Limit:  testCase.request.Limit,
						Offset: testCase.request.Offset,
						Roles:  testCase.request.Roles,
					}).
					Return(testCase.repositoryMock.resp, testCase.repositoryMock.err)
			}

			service := services.NewCredentialsList(repository)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repository.AssertExpectations(t)
		})
	}
}
