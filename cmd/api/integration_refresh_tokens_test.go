package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api/codegen"
)

// STORY: The user can update its token, and the new token is valid.

func TestRefreshTokensAPI(t *testing.T) {
	t.Parallel()

	t.Run("Anon", func(t *testing.T) {
		t.Parallel()

		client, securityClient, err := getServerClient()
		require.NoError(t, err)

		token := authAnon(t, client)
		securityClient.SetToken(token)

		t.Log("CheckSession/Authenticated")
		{
			_, err = client.CheckSession(t.Context())
			require.NoError(t, err)
		}

		t.Log("GenerateRefreshToken/NoAnon")
		{
			rawRes, err := client.CreateRefreshToken(t.Context())
			require.NoError(t, err)

			require.IsType(t, &codegen.ForbiddenError{}, rawRes)
		}
	})

	t.Run("User", func(t *testing.T) {
		t.Parallel()

		client, securityClient, err := getServerClient()
		require.NoError(t, err)

		token := authAnon(t, client)
		securityClient.SetToken(token)

		user := createUser(t, client)
		securityClient.SetToken(user.token)

		claims := checkSession(t, client)
		userID := claims.GetUserID()

		var refreshToken string

		t.Log("GenerateRefreshToken")
		{
			rawRes, err := client.CreateRefreshToken(t.Context())
			require.NoError(t, err)

			res, ok := rawRes.(*codegen.RefreshToken)
			require.True(t, ok)

			refreshToken = res.GetRefreshToken()
			require.NotEmpty(t, refreshToken)
		}

		t.Log("RefreshToken")
		{
			rawRes, err := client.RefreshSession(t.Context(), codegen.RefreshSessionParams{
				RefreshToken: refreshToken,
				AccessToken:  securityClient.GetToken(),
			})
			require.NoError(t, err)

			res, ok := rawRes.(*codegen.Token)
			require.True(t, ok)
			require.NotEmpty(t, res.GetAccessToken())
			require.NotEqual(t, refreshToken, res.GetAccessToken())

			securityClient.SetToken(res.GetAccessToken())
		}

		// Claims are carried through the refresh operation.
		claims = checkSession(t, client)
		require.Equal(t, userID, claims.GetUserID())

		t.Log("GenerateRefreshToken/NotTwice")
		{
			rawRes, err := client.CreateRefreshToken(t.Context())
			require.NoError(t, err)

			require.IsType(t, &codegen.ForbiddenError{}, rawRes)
		}
	})
}
