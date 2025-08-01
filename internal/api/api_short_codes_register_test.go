package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	apimocks "github.com/a-novel/service-authentication/internal/api/mocks"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
)

func TestRequestRegistration(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type requestRegistrationData struct {
		err error
	}

	testCases := []struct {
		name string

		form *apimodels.RequestRegistrationForm

		requestRegistrationData *requestRegistrationData

		expect    apimodels.RequestRegistrationRes
		expectErr error
	}{
		{
			name: "Success",

			form: &apimodels.RequestRegistrationForm{
				Email: "user@provider.com",
			},

			requestRegistrationData: &requestRegistrationData{},

			expect: &apimodels.RequestRegistrationNoContent{},
		},
		{
			name: "Success/Lang",

			form: &apimodels.RequestRegistrationForm{
				Email: "user@provider.com",
				Lang:  apimodels.OptLang{Value: apimodels.LangFr, Set: true},
			},

			requestRegistrationData: &requestRegistrationData{},

			expect: &apimodels.RequestRegistrationNoContent{},
		},
		{
			name: "RequestRegistrationError",

			form: &apimodels.RequestRegistrationForm{
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
