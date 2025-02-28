package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/configurator/utilstest"

	"github.com/a-novel/authentication/api/codegen"
)

// STORY: user forgot its password and wants to reset it.

func TestResetPasswordAPI(t *testing.T) {
	t.Parallel()

	client, securityClient, err := getServerClient()
	require.NoError(t, err)

	anonToken := authAnon(t, client)
	securityClient.SetToken(anonToken)

	user := createUser(t, client)
	securityClient.SetToken(user.token)

	claims := checkSession(t, client)
	userID := claims.GetUserID()

	// Revert the session to anonymous.
	securityClient.SetToken(anonToken)

	newPassword := getRandomString()

	var shortCode string

	t.Run("RequestPasswordReset", func(t *testing.T) {
		capturer := utilstest.WaitForLog(logs, captureEmailLog(t, user.email), 10*time.Second)

		rawRes, err := client.RequestPasswordReset(t.Context(), &codegen.RequestPasswordResetForm{
			Email: codegen.Email(user.email),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.RequestPasswordResetNoContent{}, rawRes)

		log, err := capturer()
		require.NoError(t, err)

		// Check if the email contains the invitation code.
		shortCode, err = extractShortCode(log)
		require.NoError(t, err)
		require.NotEmpty(t, shortCode)
	})

	t.Run("ResetPassword/WrongUserID", func(t *testing.T) {
		rawRes, err := client.ResetPassword(t.Context(), &codegen.ResetPasswordForm{
			ShortCode: codegen.ShortCode(shortCode),
			Password:  codegen.Password(newPassword),
			UserID:    codegen.UserID(uuid.New()),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.ForbiddenError{}, rawRes)
	})

	t.Run("ResetPassword", func(t *testing.T) {
		rawRes, err := client.ResetPassword(t.Context(), &codegen.ResetPasswordForm{
			ShortCode: codegen.ShortCode(shortCode),
			Password:  codegen.Password(newPassword),
			UserID:    codegen.UserID(userID.Value),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.ResetPasswordNoContent{}, rawRes)
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
