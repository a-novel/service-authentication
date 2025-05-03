package api_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func TestCreateRefreshToken(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type issueRefreshTokenData struct {
		resp string
		err  error
	}

	testCases := []struct {
		name string

		issueRefreshTokenData *issueRefreshTokenData

		expect    codegen.CreateRefreshTokenRes
		expectErr error
	}{
		{
			name: "Success",

			issueRefreshTokenData: &issueRefreshTokenData{
				resp: "refresh-token",
			},

			expect: &codegen.RefreshToken{RefreshToken: "refresh-token"},
		},
		{
			name: "ErrRefreshRefreshToken",

			issueRefreshTokenData: &issueRefreshTokenData{
				err: services.ErrRefreshRefreshToken,
			},

			expect: &codegen.ForbiddenError{Error: "invalid access token"},
		},
		{
			name: "ErrRefreshTokenWithAnonSession",

			issueRefreshTokenData: &issueRefreshTokenData{
				err: services.ErrRefreshTokenWithAnonSession,
			},

			expect: &codegen.ForbiddenError{Error: "invalid access token"},
		},
		{
			name: "Error",

			issueRefreshTokenData: &issueRefreshTokenData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.WithValue(t.Context(), api.ClaimsAPIKey{}, &models.AccessTokenClaims{})

			source := apimocks.NewMockIssueRefreshTokenService(t)

			if testCase.issueRefreshTokenData != nil {
				source.EXPECT().
					IssueRefreshToken(ctx, services.IssueRefreshTokenRequest{Claims: &models.AccessTokenClaims{}}).
					Return(testCase.issueRefreshTokenData.resp, testCase.issueRefreshTokenData.err)
			}

			handler := api.API{IssueRefreshTokenService: source}

			res, err := handler.CreateRefreshToken(ctx)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
