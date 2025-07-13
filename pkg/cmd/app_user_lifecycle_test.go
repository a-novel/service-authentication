package cmdpkg_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

func testAppUserLifecycle(ctx context.Context, t *testing.T, appConfig TestConfig) {
	t.Helper()

	security := pkg.NewBearerSource()
	client, err := pkg.NewAPIClient(ctx, fmt.Sprintf("http://localhost:%v/v1", appConfig.API.Port), security)
	require.NoError(t, err)

	token := authAnon(t, appConfig)
	security.SetToken(token)

	user := createUser(t, appConfig)
	security.SetToken(user.token)

	claims := checkSession(t, client)
	userID := claims.GetUserID()

	t.Log("Login/WrongPassword")
	{
		res, err := client.CreateSession(t.Context(), &apimodels.LoginForm{
			Email:    apimodels.Email(user.email),
			Password: "fakepassword",
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.ForbiddenError{}, res)
	}

	t.Log("Login")
	{
		res, err := client.CreateSession(t.Context(), &apimodels.LoginForm{
			Email:    apimodels.Email(user.email),
			Password: apimodels.Password(user.password),
		})
		require.NoError(t, err)

		token, ok := res.(*apimodels.Token)
		require.True(t, ok, res)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	}

	security.SetToken(user.token)

	claims = checkSession(t, client)
	require.Equal(t, userID, claims.GetUserID())
}
