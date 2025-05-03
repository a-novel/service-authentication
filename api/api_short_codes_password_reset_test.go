package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func TestRequestPasswordReset(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type requestPasswordResetData struct {
		err error
	}

	testCases := []struct {
		name string

		form *codegen.RequestPasswordResetForm

		requestPasswordResetData *requestPasswordResetData

		expect    codegen.RequestPasswordResetRes
		expectErr error
	}{
		{
			name: "Success",

			form: &codegen.RequestPasswordResetForm{
				Email: "user@provider.com",
			},

			requestPasswordResetData: &requestPasswordResetData{},

			expect: &codegen.RequestPasswordResetNoContent{},
		},
		{
			name: "Success/Lang",

			form: &codegen.RequestPasswordResetForm{
				Email: "user@provider.com",
				Lang:  codegen.OptLang{Value: codegen.LangFr, Set: true},
			},

			requestPasswordResetData: &requestPasswordResetData{},

			expect: &codegen.RequestPasswordResetNoContent{},
		},
		{
			name: "RequestPasswordResetError",

			form: &codegen.RequestPasswordResetForm{
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
					RequestPasswordReset(t.Context(), services.RequestPasswordResetRequest{
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
