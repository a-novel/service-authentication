package api_test

import (
	"errors"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func TestRequestRegistration(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type requestRegistrationData struct {
		err error
	}

	testCases := []struct {
		name string

		form *codegen.RequestRegistrationForm

		requestRegistrationData *requestRegistrationData

		expect    codegen.RequestRegistrationRes
		expectErr error
	}{
		{
			name: "Success",

			form: &codegen.RequestRegistrationForm{
				Email: "user@provider.com",
			},

			requestRegistrationData: &requestRegistrationData{},

			expect: &codegen.RequestRegistrationNoContent{},
		},
		{
			name: "Success/Lang",

			form: &codegen.RequestRegistrationForm{
				Email: "user@provider.com",
				Lang:  codegen.OptLang{Value: codegen.LangFr, Set: true},
			},

			requestRegistrationData: &requestRegistrationData{},

			expect: &codegen.RequestRegistrationNoContent{},
		},
		{
			name: "RequestRegistrationError",

			form: &codegen.RequestRegistrationForm{
				Email: "user@provider.com",
			},

			requestRegistrationData: &requestRegistrationData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockRequestRegisterService(t)

			if testCase.requestRegistrationData != nil {
				source.EXPECT().
					RequestRegister(mock.Anything, services.RequestRegisterRequest{
						Email: string(testCase.form.GetEmail()),
						Lang:  models.Lang(testCase.form.GetLang().Value),
					}).
					Return(nil, testCase.requestRegistrationData.err)
			}

			handler := api.API{RequestRegisterService: source}

			res, err := handler.RequestRegistration(t.Context(), testCase.form)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
