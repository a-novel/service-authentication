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

func TestRequestPasswordReset(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type requestPasswordResetData struct {
		err error
	}

	testCases := []struct {
		name string

		form *apimodels.RequestPasswordResetForm

		requestPasswordResetData *requestPasswordResetData

		expect    apimodels.RequestPasswordResetRes
		expectErr error
	}{
		{
			name: "Success",

			form: &apimodels.RequestPasswordResetForm{
				Email: "user@provider.com",
			},

			requestPasswordResetData: &requestPasswordResetData{},

			expect: &apimodels.RequestPasswordResetNoContent{},
		},
		{
			name: "Success/Lang",

			form: &apimodels.RequestPasswordResetForm{
				Email: "user@provider.com",
				Lang:  apimodels.OptLang{Value: apimodels.LangFr, Set: true},
			},

			requestPasswordResetData: &requestPasswordResetData{},

			expect: &apimodels.RequestPasswordResetNoContent{},
		},
		{
			name: "RequestPasswordResetError",

			form: &apimodels.RequestPasswordResetForm{
				Email: "user@provider.com",
			},

			requestPasswordResetData: &requestPasswordResetData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockRequestPasswordResetService(t)

			if testCase.requestPasswordResetData != nil {
				source.EXPECT().
					RequestPasswordReset(mock.Anything, services.RequestPasswordResetRequest{
						Email: string(testCase.form.GetEmail()),
						Lang:  models.Lang(testCase.form.GetLang().Value),
					}).
					Return(nil, testCase.requestPasswordResetData.err)
			}

			handler := api.API{RequestPasswordResetService: source}

			res, err := handler.RequestPasswordReset(t.Context(), testCase.form)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
