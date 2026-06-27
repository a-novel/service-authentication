package core_test

import (
	"errors"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-json-keys/v2/pkg/go"

	"github.com/a-novel-kit/golib/grpcf"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
)

func TestTokenCreateAnon(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type issueTokenMock struct {
		resp *servicejsonkeys.ClaimsSignResponse
		err  error
	}

	testCases := []struct {
		name string

		issueTokenMock *issueTokenMock

		expect    *core.Token
		expectErr error
	}{
		{
			name: "Success",

			issueTokenMock: &issueTokenMock{
				resp: &servicejsonkeys.ClaimsSignResponse{
					Token: "access-token",
				},
			},

			expect: &core.Token{AccessToken: "access-token"},
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

			signClaimsService := coremocks.NewMockTokenCreateAnonSignClaimsService(t)

			if testCase.issueTokenMock != nil {
				signClaimsService.EXPECT().
					ClaimsSign(mock.Anything, &servicejsonkeys.ClaimsSignRequest{
						Usage: servicejsonkeys.KeyUsageAuth,
						Payload: lo.Must(grpcf.MarshalJSONAsAny(core.AccessTokenClaims{
							Roles: []string{config.RoleAnon},
						})),
					}).
					Return(testCase.issueTokenMock.resp, testCase.issueTokenMock.err)
			}

			service := core.NewTokenCreateAnon(signClaimsService)

			resp, err := service.Exec(ctx)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			signClaimsService.AssertExpectations(t)
		})
	}
}
