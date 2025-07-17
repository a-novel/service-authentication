package cmdpkg_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

func testAppResetPassword(ctx context.Context, t *testing.T, appConfig TestConfig) {
	t.Helper()

	security := pkg.NewBearerSource()
	client, err := pkg.NewAPIClient(ctx, fmt.Sprintf("http://localhost:%v/v1", appConfig.API.Port), security)
	require.NoError(t, err)

	anonToken := authAnon(t, appConfig)
	security.SetToken(anonToken)

	user := createUser(t, appConfig)
	security.SetToken(user.token)

	claims := checkSession(t, client)
	userID := claims.GetUserID()

	// Revert the session to anonymous.
	security.SetToken(anonToken)

	newPassword := rand.Text()

	var shortCode string

	t.Log("RequestPasswordReset")
	{
		rawRes, err := client.RequestPasswordReset(t.Context(), &apimodels.RequestPasswordResetForm{
			Email: apimodels.Email(user.email),
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.RequestPasswordResetNoContent{}, rawRes)

		var ok bool

		require.Eventually(t, func() bool {
			shortCode, ok = getShortCode(user.id, "password-reset", appConfig)

			return assert.True(t, ok)
		}, 10*time.Second, 100*time.Millisecond)
	}

	t.Log("ResetPassword/WrongUserID")
	{
		rawRes, err := client.ResetPassword(t.Context(), &apimodels.ResetPasswordForm{
			ShortCode: apimodels.ShortCode(shortCode),
			Password:  apimodels.Password(newPassword),
			UserID:    apimodels.UserID(uuid.New()),
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.ForbiddenError{}, rawRes)
	}

	t.Log("ResetPassword")
	{
		rawRes, err := client.ResetPassword(t.Context(), &apimodels.ResetPasswordForm{
			ShortCode: apimodels.ShortCode(shortCode),
			Password:  apimodels.Password(newPassword),
			UserID:    apimodels.UserID(userID.Value),
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.ResetPasswordNoContent{}, rawRes)
	}

	t.Log("LoginWithOldPasswordKO")
	{
		res, err := client.CreateSession(t.Context(), &apimodels.LoginForm{
			Email:    apimodels.Email(user.email),
			Password: apimodels.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.ForbiddenError{}, res)
	}

	t.Log("LoginWithNewPasswordOK")
	{
		res, err := client.CreateSession(t.Context(), &apimodels.LoginForm{
			Email:    apimodels.Email(user.email),
			Password: apimodels.Password(newPassword),
		})
		require.NoError(t, err)

		token, ok := res.(*apimodels.Token)
		require.True(t, ok, res)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	}
}
