package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api/codegen"
)

// STORY: The user can update its password.

func TestUpdatePasswordAPI(t *testing.T) {
	t.Parallel()

	client, securityClient, err := getServerClient()
	require.NoError(t, err)

	anonToken := authAnon(t, client)
	securityClient.SetToken(anonToken)

	user := createUser(t, client)
	securityClient.SetToken(user.token)

	newPassword := getRandomString()

	t.Run("UpdatePassword/WrongPassword", func(t *testing.T) {
		rawRes, err := client.UpdatePassword(t.Context(), &codegen.UpdatePasswordForm{
			Email:           codegen.Email(user.email),
			Password:        codegen.Password(newPassword),
			CurrentPassword: "fakepassword",
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.ForbiddenError{}, rawRes)
	})

	t.Run("UpdatePassword", func(t *testing.T) {
		rawRes, err := client.UpdatePassword(t.Context(), &codegen.UpdatePasswordForm{
			Email:           codegen.Email(user.email),
			Password:        codegen.Password(newPassword),
			CurrentPassword: codegen.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.UpdatePasswordNoContent{}, rawRes)
	})

	t.Run("LoginWithOldPasswordKO", func(t *testing.T) {
		res, err := client.CreateSession(t.Context(), &codegen.LoginForm{
			Email:    codegen.Email(user.email),
			Password: codegen.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.ForbiddenError{}, res)
	})

	t.Run("LoginWithNewPasswordOK", func(t *testing.T) {
		res, err := client.CreateSession(t.Context(), &codegen.LoginForm{
			Email:    codegen.Email(user.email),
			Password: codegen.Password(newPassword),
		})
		require.NoError(t, err)

		token, ok := res.(*codegen.Token)
		require.True(t, ok)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	})
}
