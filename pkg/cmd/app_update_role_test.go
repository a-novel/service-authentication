package cmdpkg_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

func testAppUpdateRole(ctx context.Context, t *testing.T, appConfig TestConfig) {
	t.Helper()

	security := pkg.NewBearerSource()
	client, err := pkg.NewAPIClient(ctx, fmt.Sprintf("http://localhost:%v/v1", appConfig.API.Port), security)
	require.NoError(t, err)

	anonToken := authAnon(t, appConfig)
	security.SetToken(anonToken)

	user1 := createUser(t, appConfig)
	security.SetToken(user1.token)

	user1Claims := checkSession(t, client)

	user2 := createUser(t, appConfig)
	security.SetToken(user2.token)

	user2Claims := checkSession(t, client)

	t.Log("NotEnoughPrivileges")
	{
		rawRes, err := client.UpdateRole(t.Context(), &apimodels.UpdateRoleForm{
			UserID: apimodels.UserID(user1Claims.GetUserID().Value),
			Role:   apimodels.CredentialsRoleAdmin,
		})

		require.NoError(t, err)

		_, ok := rawRes.(*apimodels.ForbiddenError)
		require.True(t, ok, rawRes)
	}

	// Elevate user2 to super_admin.
	{
		updateRoleDAO := dao.NewUpdateCredentialsRoleRepository()

		_, err = updateRoleDAO.UpdateCredentialsRole(
			ctx, user2Claims.GetUserID().Value, dao.UpdateCredentialsRoleData{
				Role: models.CredentialsRoleSuperAdmin,
				Now:  time.Now(),
			},
		)
		require.NoError(t, err)

		t.Log("RefreshToken")
		{
			rawRes, err := client.RefreshSession(t.Context(), apimodels.RefreshSessionParams{
				RefreshToken: user2.refreshToken,
				AccessToken:  security.GetToken(),
			})
			require.NoError(t, err)

			res, ok := rawRes.(*apimodels.Token)
			require.True(t, ok, rawRes)

			security.SetToken(res.GetAccessToken())
		}
	}

	t.Log("OK")
	{
		rawRes, err := client.UpdateRole(t.Context(), &apimodels.UpdateRoleForm{
			UserID: apimodels.UserID(user1Claims.GetUserID().Value),
			Role:   apimodels.CredentialsRoleAdmin,
		})
		require.NoError(t, err)

		res, ok := rawRes.(*apimodels.User)
		require.True(t, ok, rawRes)
		require.Equal(t, apimodels.CredentialsRoleAdmin, res.GetRole())
	}
}
