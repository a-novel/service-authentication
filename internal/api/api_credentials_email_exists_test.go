package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	apimocks "github.com/a-novel/service-authentication/internal/api/mocks"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models/api"
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

		params apimodels.EmailExistsParams

		emailExistsData *emailExistsData

		expect    apimodels.EmailExistsRes
		expectErr error
	}{
		{
			name: "Success",

			params: apimodels.EmailExistsParams{
				Email: "user@provider.com",
			},

			emailExistsData: &emailExistsData{
				resp: true,
			},

			expect: &apimodels.EmailExistsNoContent{},
		},
		{
			name: "EmailNotFound",

			params: apimodels.EmailExistsParams{
				Email: "user@provider.com",
			},

			emailExistsData: &emailExistsData{
				resp: false,
			},

			expect: &apimodels.NotFoundError{Error: "email not found"},
		},
		{
			name: "EmailNotFoundError",

			params: apimodels.EmailExistsParams{
				Email: "user@provider.com",
			},

			emailExistsData: &emailExistsData{
				err: dao.ErrCredentialsNotFound,
			},

			expect: &apimodels.NotFoundError{Error: "email not found"},
		},
		{
			name: "Error",

			params: apimodels.EmailExistsParams{
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
					EmailExists(mock.Anything, services.EmailExistsRequest{
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
