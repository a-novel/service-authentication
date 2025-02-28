package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/configurator/utilstest"

	"github.com/a-novel/authentication/api/codegen"
)

// STORY: The user can update its email.

func TestUpdateEmailAPI(t *testing.T) {
	t.Parallel()

	client, securityClient, err := getServerClient()
	require.NoError(t, err)

	anonToken := authAnon(t, client)
	securityClient.SetToken(anonToken)

	user := createUser(t, client)
	securityClient.SetToken(user.token)

	claims := checkSession(t, client)
	userID := claims.GetUserID()

	var shortCode string

	newEmail := getRandomString() + "@example.com"

	t.Run("RequestEmailUpdate", func(t *testing.T) {
		capturer := utilstest.WaitForLog(logs, captureEmailLog(t, newEmail), 10*time.Second)

		rawRes, err := client.RequestEmailUpdate(t.Context(), &codegen.RequestEmailUpdateForm{
			Email: codegen.Email(newEmail),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.RequestEmailUpdateNoContent{}, rawRes)

		log, err := capturer()
		require.NoError(t, err)

		// Check if the email contains the invitation code.
		shortCode, err = extractShortCode(log)
		require.NoError(t, err)
		require.NotEmpty(t, shortCode)
	})

	// Following operations are expected to be performed anonymously.
	securityClient.SetToken(anonToken)

	t.Run("UpdateEmail/WrongUserID", func(t *testing.T) {
		rawRes, err := client.UpdateEmail(t.Context(), &codegen.UpdateEmailForm{
			ShortCode: codegen.ShortCode(shortCode),
			UserID:    codegen.UserID(uuid.New()),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.ForbiddenError{}, rawRes)
	})

	t.Run("UpdateEmail", func(t *testing.T) {
		rawRes, err := client.UpdateEmail(t.Context(), &codegen.UpdateEmailForm{
			ShortCode: codegen.ShortCode(shortCode),
			UserID:    codegen.UserID(userID.Value),
		})
		require.NoError(t, err)

		res, ok := rawRes.(*codegen.UpdateEmailOK)
		require.True(t, ok)

		require.NotEmpty(t, res.GetEmail())
		require.Equal(t, codegen.Email(newEmail), res.GetEmail())
	})

	t.Run("LoginWithOldEmailKO", func(t *testing.T) {
		res, err := client.CreateSession(t.Context(), &codegen.LoginForm{
			Email:    codegen.Email(user.email),
			Password: codegen.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.NotFoundError{}, res)
	})

	t.Run("LoginWithNewEmailOK", func(t *testing.T) {
		res, err := client.CreateSession(t.Context(), &codegen.LoginForm{
			Email:    codegen.Email(newEmail),
			Password: codegen.Password(user.password),
		})
		require.NoError(t, err)

		token, ok := res.(*codegen.Token)
		require.True(t, ok)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	})
}
