package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	apimocks "github.com/a-novel/authentication/api/mocks"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
)

func TestCreateSession(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type loginData struct {
		resp string
		err  error
	}

	testCases := []struct {
		name string

		req *codegen.LoginForm

		loginData *loginData

		expect    codegen.CreateSessionRes
		expectErr error
	}{
		{
			name: "Success",

			req: &codegen.LoginForm{
				Email:    "user@provider.com",
				Password: "secret",
			},

			loginData: &loginData{
				resp: "access-token",
			},

			expect: &codegen.Token{AccessToken: "access-token"},
		},
		{
			name: "UserNotFound",

			req: &codegen.LoginForm{
				Email:    "user@provider.com",
				Password: "secret",
			},

			loginData: &loginData{
				err: dao.ErrCredentialsNotFound,
			},

			expect: &codegen.NotFoundError{Error: "user not found"},
		},
		{
			name: "InvalidPassword",

			req: &codegen.LoginForm{
				Email:    "user@provider.com",
				Password: "secret",
			},

			loginData: &loginData{
				err: lib.ErrInvalidPassword,
			},

			expect: &codegen.ForbiddenError{Error: "invalid user password"},
		},
		{
			name: "Error",

			req: &codegen.LoginForm{
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
				source.
					On("Login", t.Context(), mock.AnythingOfType("LoginRequest")).
					Return(testCase.loginData.resp, testCase.loginData.err)
			}

			handler := api.API{LoginService: source}

			res, err := handler.CreateSession(t.Context(), testCase.req)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
