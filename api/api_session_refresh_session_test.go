package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	apimocks "github.com/a-novel/authentication/api/mocks"
	"github.com/a-novel/authentication/internal/services"
	"github.com/a-novel/authentication/models"
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

		params codegen.RefreshSessionParams

		consumeRefreshTokenData *consumeRefreshTokenData

		expect    codegen.RefreshSessionRes
		expectErr error
	}{
		{
			name: "Success",

			params: codegen.RefreshSessionParams{
				RefreshToken: "refresh-token",
				AccessToken:  "access-token",
			},

			consumeRefreshTokenData: &consumeRefreshTokenData{
				resp: "new-access-token",
			},

			expect: &codegen.Token{AccessToken: "new-access-token"},
		},
		{
			name: "Unauthorized",

			params: codegen.RefreshSessionParams{
				RefreshToken: "refresh-token",
				AccessToken:  "access-token",
			},

			consumeRefreshTokenData: &consumeRefreshTokenData{
				err: models.ErrUnauthorized,
			},

			expect: &codegen.ForbiddenError{Error: "invalid user password"},
		},
		{
			name: "MismatchRefreshClaims",

			params: codegen.RefreshSessionParams{
				RefreshToken: "refresh-token",
				AccessToken:  "access-token",
			},

			consumeRefreshTokenData: &consumeRefreshTokenData{
				err: services.ErrMismatchRefreshClaims,
			},

			expect: &codegen.UnprocessableEntityError{Error: "invalid refresh token"},
		},
		{
			name: "TokenIssuedWithDifferentRefreshToken",

			params: codegen.RefreshSessionParams{
				RefreshToken: "refresh-token",
				AccessToken:  "access-token",
			},

			consumeRefreshTokenData: &consumeRefreshTokenData{
				err: services.ErrTokenIssuedWithDifferentRefreshToken,
			},

			expect: &codegen.UnprocessableEntityError{Error: "invalid refresh token"},
		},
		{
			name: "Error",

			params: codegen.RefreshSessionParams{
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
					ConsumeRefreshToken(t.Context(), mock.AnythingOfType("services.ConsumeRefreshTokenRequest")).
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
