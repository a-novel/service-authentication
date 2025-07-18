package cmdpkg_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/ogen"

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
		_, err = ogen.MustGetResponse[apimodels.CreateSessionRes, *apimodels.ForbiddenError](
			client.CreateSession(t.Context(), &apimodels.LoginForm{
				Email:    apimodels.Email(user.email),
				Password: "fakepassword",
			}),
		)
		require.NoError(t, err)
	}

	t.Log("Login")
	{
		token, err := ogen.MustGetResponse[apimodels.CreateSessionRes, *apimodels.Token](
			client.CreateSession(t.Context(), &apimodels.LoginForm{
				Email:    apimodels.Email(user.email),
				Password: apimodels.Password(user.password),
			}),
		)
		require.NoError(t, err)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	}

	security.SetToken(user.token)

	claims = checkSession(t, client)
	require.Equal(t, userID, claims.GetUserID())
}
