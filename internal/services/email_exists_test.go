package services_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/internal/services"
	servicesmocks "github.com/a-novel/authentication/internal/services/mocks"
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

		request services.EmailExistsRequest

		emailExistsData *emailExistsData

		expect    bool
		expectErr error
	}{
		{
			name: "Success",

			request: services.EmailExistsRequest{
				Email: "user@provider.com",
			},

			emailExistsData: &emailExistsData{
				resp: true,
			},

			expect: true,
		},
		{
			name: "Error",

			request: services.EmailExistsRequest{
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

			source := servicesmocks.NewMockEmailExistsSource(t)

			if testCase.emailExistsData != nil {
				source.
					On("ExistsCredentialsEmail", t.Context(), testCase.request.Email).
					Return(testCase.emailExistsData.resp, testCase.emailExistsData.err)
			}

			service := services.NewEmailExistsService(source)

			resp, err := service.EmailExists(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
