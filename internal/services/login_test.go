package services_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/internal/services"
	servicesmocks "github.com/a-novel/authentication/internal/services/mocks"
	"github.com/a-novel/authentication/models"
)

func TestLogin(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	passwordRaw := "password"
	passwordScrypted, err := lib.GenerateScrypt(passwordRaw, lib.ScryptParamsDefault)
	require.NoError(t, err)

	type selectCredentialsData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	type issueTokenData struct {
		resp string
		err  error
	}

	testCases := []struct {
		name string

		request services.LoginRequest

		selectCredentialsData *selectCredentialsData
		issueTokenData        *issueTokenData

		expect    string
		expectErr error
	}{
		{
			name: "Success",

			request: services.LoginRequest{Email: "user@provider.com", Password: passwordRaw},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
				},
			},

			issueTokenData: &issueTokenData{
				resp: "access-token",
			},

			expect: "access-token",
		},
		{
			name: "Error/WrongPassword",

			request: services.LoginRequest{Email: "user@provider.com", Password: "fake-password"},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
				},
			},

			expectErr: lib.ErrInvalidPassword,
		},
		{
			name: "Error/IssueToken",

			request: services.LoginRequest{Email: "user@provider.com", Password: passwordRaw},

			selectCredentialsData: &selectCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Password: passwordScrypted,
				},
			},

			issueTokenData: &issueTokenData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SelectCredentials",

			request: services.LoginRequest{Email: "user@provider.com", Password: "fake-password"},

			selectCredentialsData: &selectCredentialsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			source := servicesmocks.NewMockLoginSource(t)

			if testCase.selectCredentialsData != nil {
				source.
					On("SelectCredentialsByEmail", ctx, testCase.request.Email).
					Return(testCase.selectCredentialsData.resp, testCase.selectCredentialsData.err)
			}

			if testCase.issueTokenData != nil {
				source.
					On("IssueToken", ctx, services.IssueTokenRequest{
						UserID: &testCase.selectCredentialsData.resp.ID,
						Roles:  []models.Role{models.RoleUser},
					}).
					Return(testCase.issueTokenData.resp, testCase.issueTokenData.err)
			}

			service := services.NewLoginService(source)

			resp, err := service.Login(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
