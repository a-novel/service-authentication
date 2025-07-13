package cmdpkg_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

func testAppUpdatePassword(ctx context.Context, t *testing.T, appConfig TestConfig) {
	t.Helper()

	security := pkg.NewBearerSource()
	client, err := pkg.NewAPIClient(ctx, fmt.Sprintf("http://localhost:%v/v1", appConfig.API.Port), security)
	require.NoError(t, err)

	anonToken := authAnon(t, appConfig)
	security.SetToken(anonToken)

	user := createUser(t, appConfig)
	security.SetToken(user.token)

	newPassword := rand.Text()

	t.Log("UpdatePassword/WrongPassword")
	{
		rawRes, err := client.UpdatePassword(t.Context(), &apimodels.UpdatePasswordForm{
			Password:        apimodels.Password(newPassword),
			CurrentPassword: "fakepassword",
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.ForbiddenError{}, rawRes)
	}

	t.Log("UpdatePassword")
	{
		rawRes, err := client.UpdatePassword(t.Context(), &apimodels.UpdatePasswordForm{
			Password:        apimodels.Password(newPassword),
			CurrentPassword: apimodels.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.UpdatePasswordNoContent{}, rawRes)
	}

	t.Log("LoginWithOldPasswordKO")
	{
		res, err := client.CreateSession(t.Context(), &apimodels.LoginForm{
			Email:    apimodels.Email(user.email),
			Password: apimodels.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.ForbiddenError{}, res)
	}

	t.Log("LoginWithNewPasswordOK")
	{
		res, err := client.CreateSession(t.Context(), &apimodels.LoginForm{
			Email:    apimodels.Email(user.email),
			Password: apimodels.Password(newPassword),
		})
		require.NoError(t, err)

		token, ok := res.(*apimodels.Token)
		require.True(t, ok, res)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	}
}
