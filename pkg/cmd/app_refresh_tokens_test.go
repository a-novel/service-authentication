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

func testAppRefreshToken(ctx context.Context, t *testing.T, appConfig TestConfig) {
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

	t.Log("RefreshToken")
	{
		res, err := ogen.MustGetResponse[apimodels.RefreshSessionRes, *apimodels.Token](
			client.RefreshSession(t.Context(), apimodels.RefreshSessionParams{
				RefreshToken: user.refreshToken,
				AccessToken:  user.token,
			}),
		)
		require.NoError(t, err)

		require.NotEmpty(t, res.GetAccessToken())
		require.NotEqual(t, res.GetRefreshToken(), res.GetAccessToken())

		security.SetToken(res.GetAccessToken())
	}

	// Claims are carried through the refresh operation.
	claims = checkSession(t, client)
	require.Equal(t, userID, claims.GetUserID())
}
