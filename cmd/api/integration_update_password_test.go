package main

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api/apiclient/testapiclient"
	"github.com/a-novel/service-authentication/api/codegen"
)

// STORY: The user can update its password.

func TestUpdatePasswordAPI(t *testing.T) {
	client, securityClient, err := testapiclient.GetServerClient()
	require.NoError(t, err)

	anonToken := authAnon(t, client)
	securityClient.SetToken(anonToken)

	user := createUser(t, client)
	securityClient.SetToken(user.token)

	newPassword := rand.Text()

	t.Log("UpdatePassword/WrongPassword")
	{
		rawRes, err := client.UpdatePassword(t.Context(), &codegen.UpdatePasswordForm{
			Password:        codegen.Password(newPassword),
			CurrentPassword: "fakepassword",
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.ForbiddenError{}, rawRes)
	}

	t.Log("UpdatePassword")
	{
		rawRes, err := client.UpdatePassword(t.Context(), &codegen.UpdatePasswordForm{
			Password:        codegen.Password(newPassword),
			CurrentPassword: codegen.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.UpdatePasswordNoContent{}, rawRes)
	}

	t.Log("LoginWithOldPasswordKO")
	{
		res, err := client.CreateSession(t.Context(), &codegen.LoginForm{
			Email:    codegen.Email(user.email),
			Password: codegen.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.ForbiddenError{}, res)
	}

	t.Log("LoginWithNewPasswordOK")
	{
		res, err := client.CreateSession(t.Context(), &codegen.LoginForm{
			Email:    codegen.Email(user.email),
			Password: codegen.Password(newPassword),
		})
		require.NoError(t, err)

		token, ok := res.(*codegen.Token)
		require.True(t, ok, res)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	}
}
