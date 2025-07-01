package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api/apiclient/testapiclient"
	"github.com/a-novel/service-authentication/api/codegen"
)

// STORY: The user can create an anonymous session, and this session is valid.

func authAnon(t *testing.T, client *codegen.Client) string {
	t.Helper()

	var token string

	t.Log("Authenticate/Anon")
	{
		rawRes, err := client.CreateAnonSession(t.Context())
		require.NoError(t, err)

		res, ok := rawRes.(*codegen.Token)
		require.True(t, ok, rawRes)
		require.NotEmpty(t, res.GetAccessToken())

		token = res.GetAccessToken()
	}

	return token
}

func checkSession(t *testing.T, client *codegen.Client) *codegen.Claims {
	t.Helper()

	rawRes, err := client.CheckSession(t.Context())
	require.NoError(t, err)

	res, ok := rawRes.(*codegen.Claims)
	require.True(t, ok, rawRes)

	return res
}

func TestAnonAuthAPI(t *testing.T) {
	client, securityClient, err := testapiclient.GetServerClient()
	require.NoError(t, err)

	t.Log("CheckSession/Unauthenticated")
	{
		rawRes, err := client.CheckSession(t.Context())

		require.NoError(t, err)

		_, ok := rawRes.(*codegen.UnauthorizedError)
		require.True(t, ok, rawRes)
	}

	token := authAnon(t, client)
	securityClient.SetToken(token)

	_ = checkSession(t, client)
}
