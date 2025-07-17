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

func testAppUpdateEmail(ctx context.Context, t *testing.T, appConfig TestConfig) {
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

	var shortCode string

	newEmail := rand.Text() + "@example.com"

	t.Log("RequestEmailUpdate")
	{
		rawRes, err := client.RequestEmailUpdate(t.Context(), &apimodels.RequestEmailUpdateForm{
			Email: apimodels.Email(newEmail),
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.RequestEmailUpdateNoContent{}, rawRes)

		var ok bool

		require.Eventually(t, func() bool {
			shortCode, ok = getShortCode(user.id, "email-update", appConfig)

			return assert.True(t, ok)
		}, 10*time.Second, 100*time.Millisecond)
	}

	// Following operations are expected to be performed anonymously.
	security.SetToken(anonToken)

	t.Log("UpdateEmail/WrongUserID")
	{
		rawRes, err := client.UpdateEmail(t.Context(), &apimodels.UpdateEmailForm{
			ShortCode: apimodels.ShortCode(shortCode),
			UserID:    apimodels.UserID(uuid.New()),
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.ForbiddenError{}, rawRes)
	}

	t.Log("UpdateEmail")
	{
		rawRes, err := client.UpdateEmail(t.Context(), &apimodels.UpdateEmailForm{
			ShortCode: apimodels.ShortCode(shortCode),
			UserID:    apimodels.UserID(userID.Value),
		})
		require.NoError(t, err)

		res, ok := rawRes.(*apimodels.NewEmail)
		require.True(t, ok, res)

		require.NotEmpty(t, res)
		require.Equal(t, apimodels.Email(newEmail), res.Email)
	}

	t.Log("LoginWithOldEmailKO")
	{
		res, err := client.CreateSession(t.Context(), &apimodels.LoginForm{
			Email:    apimodels.Email(user.email),
			Password: apimodels.Password(user.password),
		})
		require.NoError(t, err)

		require.IsType(t, &apimodels.NotFoundError{}, res)
	}

	t.Log("LoginWithNewEmailOK")
	{
		res, err := client.CreateSession(t.Context(), &apimodels.LoginForm{
			Email:    apimodels.Email(newEmail),
			Password: apimodels.Password(user.password),
		})
		require.NoError(t, err)

		token, ok := res.(*apimodels.Token)
		require.True(t, ok, res)

		require.NotEqual(t, token.GetAccessToken(), user.token)
		user.token = token.GetAccessToken()
	}
}
