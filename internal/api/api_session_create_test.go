package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	apimocks "github.com/a-novel/service-authentication/internal/api/mocks"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

func TestCreateSession(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type loginData struct {
		resp *models.Token
		err  error
	}

	testCases := []struct {
		name string

		form *apimodels.LoginForm

		loginData *loginData

		expect    apimodels.CreateSessionRes
		expectErr error
	}{
		{
			name: "Success",

			form: &apimodels.LoginForm{
				Email:    "user@provider.com",
				Password: "secret",
			},

			loginData: &loginData{
				resp: &models.Token{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
				},
			},

			expect: &apimodels.Token{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			},
		},
		{
			name: "UserNotFound",

			form: &apimodels.LoginForm{
				Email:    "user@provider.com",
				Password: "secret",
			},

			loginData: &loginData{
				err: dao.ErrCredentialsNotFound,
			},

			expect: &apimodels.NotFoundError{Error: "user not found"},
		},
		{
			name: "InvalidPassword",

			form: &apimodels.LoginForm{
				Email:    "user@provider.com",
				Password: "secret",
			},

			loginData: &loginData{
				err: lib.ErrInvalidPassword,
			},

			expect: &apimodels.ForbiddenError{Error: "invalid user password"},
		},
		{
			name: "Error",

			form: &apimodels.LoginForm{
				Email:    "user@provider.com",
				Password: "secret",
			},

			loginData: &loginData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockLoginService(t)

			if testCase.loginData != nil {
				source.EXPECT().
					Login(mock.Anything, services.LoginRequest{
						Email:    string(testCase.form.GetEmail()),
						Password: string(testCase.form.GetPassword()),
					}).
					Return(testCase.loginData.resp, testCase.loginData.err)
			}

			handler := api.API{LoginService: source}

			res, err := handler.CreateSession(t.Context(), testCase.form)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
