package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	apimocks "github.com/a-novel/authentication/api/mocks"
)

func TestRequestPasswordReset(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type requestPasswordResetData struct {
		err error
	}

	testCases := []struct {
		name string

		req *codegen.RequestPasswordResetForm

		requestPasswordResetData *requestPasswordResetData

		expect    codegen.RequestPasswordResetRes
		expectErr error
	}{
		{
			name: "Success",

			req: &codegen.RequestPasswordResetForm{
				Email: "user@provider.com",
			},

			requestPasswordResetData: &requestPasswordResetData{},

			expect: &codegen.RequestPasswordResetNoContent{},
		},
		{
			name: "RequestPasswordResetError",

			req: &codegen.RequestPasswordResetForm{
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
				source.
					On("RequestPasswordReset", t.Context(), mock.AnythingOfType("services.RequestPasswordResetRequest")).
					Return(nil, testCase.requestPasswordResetData.err)
			}

			handler := api.API{RequestPasswordResetService: source}

			res, err := handler.RequestPasswordReset(t.Context(), testCase.req)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
