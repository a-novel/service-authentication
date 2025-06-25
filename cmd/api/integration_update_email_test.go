package main

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/configurator/utilstest"

	"github.com/a-novel/service-authentication/api/apiclient/testapiclient"
	"github.com/a-novel/service-authentication/api/codegen"
)

// STORY: The user can update its email.

func TestUpdateEmailAPI(t *testing.T) {
	client, securityClient, err := testapiclient.GetServerClient()
	require.NoError(t, err)

	anonToken := authAnon(t, client)
	securityClient.SetToken(anonToken)

	user := createUser(t, client)
	securityClient.SetToken(user.token)

	claims := checkSession(t, client)
	userID := claims.GetUserID()

	var shortCode string

	newEmail := rand.Text() + "@example.com"

	t.Log("RequestEmailUpdate")
	{
		capturer := utilstest.WaitForLog(logs, captureEmailLog(t, newEmail), 3*time.Second)

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
	}

	// Following operations are expected to be performed anonymously.
	securityClient.SetToken(anonToken)

	t.Log("UpdateEmail/WrongUserID")
	{
		rawRes, err := client.UpdateEmail(t.Context(), &codegen.UpdateEmailForm{
			ShortCode: codegen.ShortCode(shortCode),
			UserID:    codegen.UserID(uuid.New()),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.ForbiddenError{}, rawRes)
	}

	t.Log("UpdateEmail")
	{
		rawRes, err := client.UpdateEmail(t.Context(), &codegen.UpdateEmailForm{
			ShortCode: codegen.ShortCode(shortCode),
			UserID:    codegen.UserID(userID.Value),
		})
		require.NoError(t, err)

		res, ok := rawRes.(*codegen.Email)
		require.True(t, ok)

		require.NotEmpty(t, res)
		require.Equal(t, codegen.Email(newEmail), lo.FromPtr(res))
	}

	t.Log("LoginWithOldEmailKO")
	{
		res, err := client.CreateSession(t.Context(), &codegen.LoginForm{
			Email:    codegen.Email(user.email),
			Password: codegen.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &codegen.NotFoundError{}, res)
	}

	t.Log("LoginWithNewEmailOK")
	{
		res, err := client.CreateSession(t.Context(), &codegen.LoginForm{
			Email:    codegen.Email(newEmail),
			Password: codegen.Password(user.password),
		})
		require.NoError(t, err)

		token, ok := res.(*codegen.Token)
		require.True(t, ok)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	}
}
