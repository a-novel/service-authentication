package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
)

func TestCreateAnonSession(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type loginData struct {
		resp string
		err  error
	}

	testCases := []struct {
		name string

		loginData *loginData

		expect    codegen.CreateAnonSessionRes
		expectErr error
	}{
		{
			name: "Success",

			loginData: &loginData{
				resp: "access-token",
			},

			expect: &codegen.Token{AccessToken: "access-token"},
		},
		{
			name: "Error",

			loginData: &loginData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockLoginAnonService(t)

			if testCase.loginData != nil {
				source.EXPECT().
					LoginAnon(mock.Anything).
					Return(testCase.loginData.resp, testCase.loginData.err)
			}

			handler := api.API{LoginAnonService: source}

			res, err := handler.CreateAnonSession(t.Context())
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
