package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api/codegen"
)

// STORY: The user can create an anonymous session, and this session is valid.

func authAnon(t *testing.T, client *codegen.Client) string {
	t.Helper()

	var token string

	t.Run("Authenticate/Anon", func(t *testing.T) {
		rawRes, err := client.CreateAnonSession(t.Context())
		require.NoError(t, err)

		res, ok := rawRes.(*codegen.Token)
		require.True(t, ok)
		require.NotEmpty(t, res.GetAccessToken())

		token = res.GetAccessToken()
	})

	return token
}

func checkSession(t *testing.T, client *codegen.Client) *codegen.Claims {
	rawRes, err := client.CheckSession(t.Context())
	require.NoError(t, err)

	res, ok := rawRes.(*codegen.Claims)
	require.True(t, ok)

	return res
}

func TestAnonAuthAPI(t *testing.T) {
	t.Parallel()

	client, securityClient, err := getServerClient()
	require.NoError(t, err)

	t.Run("CheckSession/Unauthenticated", func(t *testing.T) {
		_, err = client.CheckSession(t.Context())

		var ogenError *codegen.UnexpectedErrorStatusCode

		require.ErrorAs(t, err, &ogenError)
		require.Equal(t, http.StatusUnauthorized, ogenError.StatusCode)
	})

	token := authAnon(t, client)
	securityClient.SetToken(token)

	_ = checkSession(t, client)
}
