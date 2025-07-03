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
	"github.com/a-novel/service-authentication/internal/api/codegen"
	apimocks "github.com/a-novel/service-authentication/internal/api/mocks"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
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

		params codegen.ListUsersParams

		listUsersData *listUsersData

		expect    codegen.ListUsersRes
		expectErr error
	}{
		{
			name: "Success",

			params: codegen.ListUsersParams{
				Limit:  codegen.OptInt{Value: 10, Set: true},
				Offset: codegen.OptInt{Value: 2, Set: true},
				Roles:  []codegen.CredentialsRole{codegen.CredentialsRoleUser},
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

			expect: &codegen.ListUsersOKApplicationJSON{
				{
					ID:        codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
					Email:     "user3@email.com",
					Role:      codegen.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
					Email:     "user2@email.com",
					Role:      codegen.CredentialsRoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Email:     "user1@email.com",
					Role:      codegen.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Error",

			params: codegen.ListUsersParams{
				Limit:  codegen.OptInt{Value: 10, Set: true},
				Offset: codegen.OptInt{Value: 2, Set: true},
				Roles:  []codegen.CredentialsRole{codegen.CredentialsRoleUser},
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
							func(item codegen.CredentialsRole, _ int) models.CredentialsRole {
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
