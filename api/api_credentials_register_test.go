package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func TestRegister(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type registerData struct {
		resp *models.Token
		err  error
	}

	testCases := []struct {
		name string

		form *codegen.RegisterForm

		registerData *registerData

		expect    codegen.RegisterRes
		expectErr error
	}{
		{
			name: "Success",

			form: &codegen.RegisterForm{
				Email:     "user@provider.com",
				Password:  "secret",
				ShortCode: "foobarqux",
			},

			registerData: &registerData{
				resp: &models.Token{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
				},
			},

			expect: &codegen.Token{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			},
		},
		{
			name: "EmailAlreadyExists",

			form: &codegen.RegisterForm{
				Email:     "user@provider.com",
				Password:  "secret",
				ShortCode: "foobarqux",
			},

			registerData: &registerData{
				err: dao.ErrCredentialsAlreadyExists,
			},

			expect: &codegen.ConflictError{Error: "email already taken"},
		},
		{
			name: "ShortCodeNotFound",

			form: &codegen.RegisterForm{
				Email:     "user@provider.com",
				Password:  "secret",
				ShortCode: "foobarqux",
			},

			registerData: &registerData{
				err: dao.ErrShortCodeNotFound,
			},

			expect: &codegen.ForbiddenError{Error: "invalid short code"},
		},
		{
			name: "InvalidShortCode",

			form: &codegen.RegisterForm{
				Email:     "user@provider.com",
				Password:  "secret",
				ShortCode: "foobarqux",
			},

			registerData: &registerData{
				err: services.ErrInvalidShortCode,
			},

			expect: &codegen.ForbiddenError{Error: "invalid short code"},
		},
		{
			name: "Error",

			form: &codegen.RegisterForm{
				Email:     "user@provider.com",
				Password:  "secret",
				ShortCode: "foobarqux",
			},

			registerData: &registerData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockRegisterService(t)

			if testCase.registerData != nil {
				source.EXPECT().
					Register(mock.Anything, services.RegisterRequest{
						Email:     string(testCase.form.GetEmail()),
						Password:  string(testCase.form.GetPassword()),
						ShortCode: string(testCase.form.GetShortCode()),
					}).
					Return(testCase.registerData.resp, testCase.registerData.err)
			}

			handler := api.API{RegisterService: source}

			res, err := handler.Register(t.Context(), testCase.form)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
