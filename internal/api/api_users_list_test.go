package api_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	apimocks "github.com/a-novel/service-authentication/internal/api/mocks"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

func TestListUsers(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type listUsersData struct {
		resp []*models.User
		err  error
	}

	testCases := []struct {
		name string

		params apimodels.ListUsersParams

		listUsersData *listUsersData

		expect    apimodels.ListUsersRes
		expectErr error
	}{
		{
			name: "Success",

			params: apimodels.ListUsersParams{
				Limit:  apimodels.OptInt{Value: 10, Set: true},
				Offset: apimodels.OptInt{Value: 2, Set: true},
				Roles:  []apimodels.CredentialsRole{apimodels.CredentialsRoleUser},
			},

			listUsersData: &listUsersData{
				resp: []*models.User{
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

			expect: &apimodels.ListUsersOKApplicationJSON{
				{
					ID:        apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
					Email:     "user3@email.com",
					Role:      apimodels.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					Email:     "user2@email.com",
					Role:      apimodels.CredentialsRoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        apimodels.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Email:     "user1@email.com",
					Role:      apimodels.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Error",

			params: apimodels.ListUsersParams{
				Limit:  apimodels.OptInt{Value: 10, Set: true},
				Offset: apimodels.OptInt{Value: 2, Set: true},
				Roles:  []apimodels.CredentialsRole{apimodels.CredentialsRoleUser},
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

			source := apimocks.NewMockListUsersService(t)

			if testCase.listUsersData != nil {
				source.EXPECT().
					ListUsers(mock.Anything, services.ListUsersRequest{
						Limit:  testCase.params.Limit.Value,
						Offset: testCase.params.Offset.Value,
						Roles: lo.Map(
							testCase.params.Roles,
							func(item apimodels.CredentialsRole, _ int) models.CredentialsRole {
								return new(api.API).CredentialsRoleToModel(item)
							},
						),
					}).
					Return(testCase.listUsersData.resp, testCase.listUsersData.err)
			}

			handler := api.API{ListUsersService: source}

			res, err := handler.ListUsers(t.Context(), testCase.params)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
