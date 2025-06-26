package api_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func TestGetUser(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type GetUserData struct {
		resp *models.User
		err  error
	}

	testCases := []struct {
		name string

		params codegen.GetUserParams

		GetUserData *GetUserData

		expect    codegen.GetUserRes
		expectErr error
	}{
		{
			name: "Success",

			params: codegen.GetUserParams{
				UserID: codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			GetUserData: &GetUserData{
				resp: &models.User{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "user1@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &codegen.User{
				ID:        codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
				Email:     "user1@email.com",
				Role:      codegen.CredentialsRoleUser,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error",

			params: codegen.GetUserParams{
				UserID: codegen.UserID(uuid.MustParse("00000000-0000-0000-0000-000000000003")),
			},

			GetUserData: &GetUserData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockGetUserService(t)

			if testCase.GetUserData != nil {
				source.EXPECT().
					SelectUser(mock.Anything, services.SelectUserRequest{
						ID: uuid.UUID(testCase.params.UserID),
					}).
					Return(testCase.GetUserData.resp, testCase.GetUserData.err)
			}

			handler := api.API{GetUserService: source}

			res, err := handler.GetUser(t.Context(), testCase.params)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
