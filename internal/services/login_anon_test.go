package services_test

import (
	"errors"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
	"github.com/a-novel/service-authentication/models"
)

func TestLoginAnon(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type issueTokenData struct {
		resp string
		err  error
	}

	testCases := []struct {
		name string

		issueTokenData *issueTokenData

		expect    string
		expectErr error
	}{
		{
			name: "Success",

			issueTokenData: &issueTokenData{
				resp: "access-token",
			},

			expect: "access-token",
		},
		{
			name: "Error/IssueToken",

			issueTokenData: &issueTokenData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			source := servicesmocks.NewMockLoginAnonSource(t)

			if testCase.issueTokenData != nil {
				source.EXPECT().
					IssueToken(mock.Anything, services.IssueTokenRequest{
						Roles: []models.Role{models.RoleAnon},
					}).
					Return(testCase.issueTokenData.resp, testCase.issueTokenData.err)
			}

			service := services.NewLoginAnonService(source)

			resp, err := service.LoginAnon(ctx)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
