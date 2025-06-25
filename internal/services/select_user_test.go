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

func TestSelectUser(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type SelectUserData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	testCases := []struct {
		name string

		request services.SelectUserRequest

		SelectUserData *SelectUserData

		expect    *models.User
		expectErr error
	}{
		{
			name: "Success",

			request: services.SelectUserRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			SelectUserData: &SelectUserData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "user1@email.com",
					Role:      models.CredentialsRoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &models.User{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "user1@email.com",
				Role:      models.CredentialsRoleUser,
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "Error",

			request: services.SelectUserRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			SelectUserData: &SelectUserData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			source := servicesmocks.NewMockSelectUserSource(t)

			if testCase.SelectUserData != nil {
				source.EXPECT().
					SelectCredentials(mock.Anything, testCase.request.ID).
					Return(testCase.SelectUserData.resp, testCase.SelectUserData.err)
			}

			service := services.NewSelectUserService(source)

			resp, err := service.SelectUser(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
