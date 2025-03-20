package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/models"
)

func TestUpdateRole(t *testing.T) {
	t.Parallel()

	client, securityClient, err := getServerClient()
	require.NoError(t, err)

	anonToken := authAnon(t, client)
	securityClient.SetToken(anonToken)

	user1 := createUser(t, client)
	securityClient.SetToken(user1.token)

	user1Claims := checkSession(t, client)

	user2 := createUser(t, client)
	securityClient.SetToken(user2.token)

	user2Claims := checkSession(t, client)

	t.Log("NotEnoughPrivileges")
	{
		_, err = client.UpdateRole(t.Context(), &codegen.UpdateRoleForm{
			UserID: codegen.UserID(user1Claims.GetUserID().Value),
			Role:   codegen.CredentialsRoleAdmin,
		})

		require.Error(t, err)
	}

	// Elevate user2 to super_admin.
	{
		ctx, err := pgctx.NewContext(t.Context(), nil)
		require.NoError(t, err)

		updateRoleDAO := dao.NewUpdateCredentialsRoleRepository()

		_, err = updateRoleDAO.UpdateCredentialsRole(
			ctx, user2Claims.GetUserID().Value, dao.UpdateCredentialsRoleData{
				Role: models.CredentialsRoleSuperAdmin,
				Now:  time.Now(),
			},
		)
		require.NoError(t, err)

		// Re-login to update token with new role.
		var refreshToken string

		t.Log("GenerateRefreshToken")
		{
			rawRes, err := client.CreateRefreshToken(t.Context())
			require.NoError(t, err)

			res, ok := rawRes.(*codegen.RefreshToken)
			require.True(t, ok)

			refreshToken = res.GetRefreshToken()
		}

		t.Log("RefreshToken")
		{
			rawRes, err := client.RefreshSession(t.Context(), codegen.RefreshSessionParams{
				RefreshToken: refreshToken,
				AccessToken:  securityClient.GetToken(),
			})
			require.NoError(t, err)

			res, ok := rawRes.(*codegen.Token)
			require.True(t, ok)

			securityClient.SetToken(res.GetAccessToken())
		}
	}

	t.Log("OK")
	{
		rawRes, err := client.UpdateRole(t.Context(), &codegen.UpdateRoleForm{
			UserID: codegen.UserID(user1Claims.GetUserID().Value),
			Role:   codegen.CredentialsRoleAdmin,
		})
		require.NoError(t, err)

		res, ok := rawRes.(*codegen.User)
		require.True(t, ok)
		require.Equal(t, codegen.CredentialsRoleAdmin, res.GetRole())
	}
}
