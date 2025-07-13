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

func TestRefreshSession(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type consumeRefreshTokenData struct {
		resp string
		err  error
	}

	testCases := []struct {
		name string

		params apimodels.RefreshSessionParams

		consumeRefreshTokenData *consumeRefreshTokenData

		expect    apimodels.RefreshSessionRes
		expectErr error
	}{
		{
			name: "Success",

			params: apimodels.RefreshSessionParams{
				RefreshToken: "refresh-token",
				AccessToken:  "access-token",
			},

			consumeRefreshTokenData: &consumeRefreshTokenData{
				resp: "new-access-token",
			},

			expect: &apimodels.Token{
				RefreshToken: "refresh-token",
				AccessToken:  "new-access-token",
			},
		},
		{
			name: "Unauthorized",

			params: apimodels.RefreshSessionParams{
				RefreshToken: "refresh-token",
				AccessToken:  "access-token",
			},

			consumeRefreshTokenData: &consumeRefreshTokenData{
				err: models.ErrUnauthorized,
			},

			expect: &apimodels.ForbiddenError{Error: "invalid user password"},
		},
		{
			name: "MismatchRefreshClaims",

			params: apimodels.RefreshSessionParams{
				RefreshToken: "refresh-token",
				AccessToken:  "access-token",
			},

			consumeRefreshTokenData: &consumeRefreshTokenData{
				err: services.ErrMismatchRefreshClaims,
			},

			expect: &apimodels.UnprocessableEntityError{Error: "invalid refresh token"},
		},
		{
			name: "TokenIssuedWithDifferentRefreshToken",

			params: apimodels.RefreshSessionParams{
				RefreshToken: "refresh-token",
				AccessToken:  "access-token",
			},

			consumeRefreshTokenData: &consumeRefreshTokenData{
				err: services.ErrTokenIssuedWithDifferentRefreshToken,
			},

			expect: &apimodels.UnprocessableEntityError{Error: "invalid refresh token"},
		},
		{
			name: "Error",

			params: apimodels.RefreshSessionParams{
				RefreshToken: "refresh-token",
				AccessToken:  "access-token",
			},

			consumeRefreshTokenData: &consumeRefreshTokenData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockConsumeRefreshTokenService(t)

			if testCase.consumeRefreshTokenData != nil {
				source.EXPECT().
					ConsumeRefreshToken(mock.Anything, services.ConsumeRefreshTokenRequest{
						AccessToken:  testCase.params.AccessToken,
						RefreshToken: testCase.params.RefreshToken,
					}).
					Return(testCase.consumeRefreshTokenData.resp, testCase.consumeRefreshTokenData.err)
			}

			handler := api.API{ConsumeRefreshTokenService: source}

			res, err := handler.RefreshSession(t.Context(), testCase.params)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
