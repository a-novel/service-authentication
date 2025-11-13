package services_test

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/grpcf"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
)

func TestTokenCreateAnon(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type issueTokenMock struct {
		resp *jkpkg.ClaimsSignResponse
		err  error
	}

	testCases := []struct {
		name string

		issueTokenMock *issueTokenMock

		expect    *services.Token
		expectErr error
	}{
		{
			name: "Success",

			issueTokenMock: &issueTokenMock{
				resp: &jkpkg.ClaimsSignResponse{
					Token: "access_token",
				},
			},

			expect: &services.Token{AccessToken: "access-token"},
		},
		{
			name: "Error/IssueToken",

			issueTokenMock: &issueTokenMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			signClaimsService := servicesmocks.NewMockTokenCreateAnonSignClaimsService(t)

			if testCase.issueTokenMock != nil {
				signClaimsService.EXPECT().
					ClaimsSign(mock.Anything, &jkpkg.ClaimsSignRequest{
						Usage: jkpkg.KeyUsageAuth,
						Payload: lo.Must(grpcf.InterfaceToProtoAny(services.AccessTokenClaims{
							Roles: []string{config.RoleAnon},
						})),
					}).
					Return(testCase.issueTokenMock.resp, testCase.issueTokenMock.err)
			}

			service := services.NewTokenCreateAnon(signClaimsService)

			resp, err := service.Exec(ctx)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			signClaimsService.AssertExpectations(t)
		})
	}
}
