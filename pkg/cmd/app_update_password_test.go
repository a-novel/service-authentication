package cmdpkg_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/ogen"

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
		_, err = ogen.MustGetResponse[apimodels.UpdatePasswordRes, *apimodels.ForbiddenError](
			client.UpdatePassword(t.Context(), &apimodels.UpdatePasswordForm{
				Password:        apimodels.Password(newPassword),
				CurrentPassword: "fakepassword",
			}),
		)
		require.NoError(t, err)
	}

	t.Log("UpdatePassword")
	{
		_, err = ogen.MustGetResponse[apimodels.UpdatePasswordRes, *apimodels.UpdatePasswordNoContent](
			client.UpdatePassword(t.Context(), &apimodels.UpdatePasswordForm{
				Password:        apimodels.Password(newPassword),
				CurrentPassword: apimodels.Password(user.password),
			}),
		)
		require.NoError(t, err)
	}

	t.Log("LoginWithOldPasswordKO")
	{
		_, err = ogen.MustGetResponse[apimodels.CreateSessionRes, *apimodels.ForbiddenError](
			client.CreateSession(t.Context(), &apimodels.LoginForm{
				Email:    apimodels.Email(user.email),
				Password: apimodels.Password(user.password),
			}),
		)
		require.NoError(t, err)
	}

	t.Log("LoginWithNewPasswordOK")
	{
		token, err := ogen.MustGetResponse[apimodels.CreateSessionRes, *apimodels.Token](
			client.CreateSession(t.Context(), &apimodels.LoginForm{
				Email:    apimodels.Email(user.email),
				Password: apimodels.Password(newPassword),
			}),
		)
		require.NoError(t, err)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	}
}
