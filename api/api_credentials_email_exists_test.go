package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
)

func TestEmailExists(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type emailExistsData struct {
		resp bool
		err  error
	}

	testCases := []struct {
		name string

		params codegen.EmailExistsParams

		emailExistsData *emailExistsData

		expect    codegen.EmailExistsRes
		expectErr error
	}{
		{
			name: "Success",

			params: codegen.EmailExistsParams{
				Email: "user@provider.com",
			},

			emailExistsData: &emailExistsData{
				resp: true,
			},

			expect: &codegen.EmailExistsNoContent{},
		},
		{
			name: "EmailNotFound",

			params: codegen.EmailExistsParams{
				Email: "user@provider.com",
			},

			emailExistsData: &emailExistsData{
				resp: false,
			},

			expect: &codegen.NotFoundError{Error: "email not found"},
		},
		{
			name: "EmailNotFoundError",

			params: codegen.EmailExistsParams{
				Email: "user@provider.com",
			},

			emailExistsData: &emailExistsData{
				err: dao.ErrCredentialsNotFound,
			},

			expect: &codegen.NotFoundError{Error: "email not found"},
		},
		{
			name: "Error",

			params: codegen.EmailExistsParams{
				Email: "user@provider.com",
			},

			emailExistsData: &emailExistsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockEmailExistsService(t)

			if testCase.emailExistsData != nil {
				source.EXPECT().
					EmailExists(t.Context(), services.EmailExistsRequest{
						Email: string(testCase.params.Email),
					}).
					Return(testCase.emailExistsData.resp, testCase.emailExistsData.err)
			}

			handler := api.API{EmailExistsService: source}

			res, err := handler.EmailExists(t.Context(), testCase.params)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
