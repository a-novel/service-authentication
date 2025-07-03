package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api/apiclient/testapiclient"
	"github.com/a-novel/service-authentication/internal/api/codegen"
)

// STORY: The user can update its token, and the new token is valid.

func TestRefreshTokensAPI(t *testing.T) {
	t.Run("User", func(t *testing.T) {
		client, securityClient, err := testapiclient.GetServerClient()
		require.NoError(t, err)

		token := authAnon(t, client)
		securityClient.SetToken(token)

		user := createUser(t, client)
		securityClient.SetToken(user.token)

		claims := checkSession(t, client)
		userID := claims.GetUserID()

		t.Log("RefreshToken")
		{
			rawRes, err := client.RefreshSession(t.Context(), codegen.RefreshSessionParams{
				RefreshToken: user.refreshToken,
				AccessToken:  user.token,
			})
			require.NoError(t, err)

			res, ok := rawRes.(*codegen.Token)
			require.True(t, ok, rawRes)
			require.NotEmpty(t, res.GetAccessToken())
			require.NotEqual(t, res.GetRefreshToken(), res.GetAccessToken())

			securityClient.SetToken(res.GetAccessToken())
		}

		// Claims are carried through the refresh operation.
		claims = checkSession(t, client)
		require.Equal(t, userID, claims.GetUserID())
	})
}
