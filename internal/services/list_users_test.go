package services_test

import (
	"errors"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestListUsers(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type listUsersData struct {
		resp []*dao.CredentialsEntity
		err  error
	}

	testCases := []struct {
		name string

		request services.ListUsersRequest

		listUsersData *listUsersData

		expect    []*models.User
		expectErr error
	}{
		{
			name: "Success",

			request: services.ListUsersRequest{
				Limit:  10,
				Offset: 0,
				Roles:  []models.CredentialsRole{models.CredentialsRoleUser},
			},

			listUsersData: &listUsersData{
				resp: []*dao.CredentialsEntity{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						Email:     "user3@email.com",
						Role:      models.CredentialsRoleUser,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						Email:     "user2@email.com",
						Role:      models.CredentialsRoleAdmin,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Email:     "user1@email.com",
						Role:      models.CredentialsRoleUser,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expect: []*models.User{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Email:     "user3@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user2@email.com",
					Role:      models.CredentialsRoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "user1@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Error",

			request: services.ListUsersRequest{
				Limit:  10,
				Offset: 0,
				Roles:  []models.CredentialsRole{models.CredentialsRoleUser},
			},

			listUsersData: &listUsersData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			source := servicesmocks.NewMockListUsersSource(t)

			if testCase.listUsersData != nil {
				source.EXPECT().
					ListUsers(mock.Anything, dao.ListUsersData{
						Limit:  testCase.request.Limit,
						Offset: testCase.request.Offset,
						Roles:  testCase.request.Roles,
					}).
					Return(testCase.listUsersData.resp, testCase.listUsersData.err)
			}

			service := services.NewListUsersService(source)

			resp, err := service.ListUsers(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
