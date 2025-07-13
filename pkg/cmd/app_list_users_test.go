package cmdpkg_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/api"
	"github.com/a-novel/service-authentication/pkg"
)

func testAppListUsers(ctx context.Context, t *testing.T, appConfig TestConfig) {
	t.Helper()

	security := pkg.NewBearerSource()
	client, err := pkg.NewAPIClient(ctx, fmt.Sprintf("http://localhost:%v/v1", appConfig.API.Port), security)
	require.NoError(t, err)

	// YOLO
	fixtures := []*dao.CredentialsEntity{
		{
			ID:        uuid.New(),
			Email:     rand.Text() + "@email.com",
			Role:      models.CredentialsRoleUser,
			CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        uuid.New(),
			Email:     rand.Text() + "@email.com",
			Role:      models.CredentialsRoleAdmin,
			CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        uuid.New(),
			Email:     rand.Text() + "@email.com",
			Role:      models.CredentialsRoleUser,
			CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
		},
	}

	tx, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	_, err = tx.NewInsert().Model(&fixtures).Exec(ctx)
	require.NoError(t, err)

	token := authAnon(t, appConfig)
	security.SetToken(token)

	// Verify the fixtures are present in the list of returned users.
	t.Log("ListUsers")
	{
		rawRes, err := client.ListUsers(ctx, apimodels.ListUsersParams{})
		require.NoError(t, err)

		res, ok := rawRes.(*apimodels.ListUsersOKApplicationJSON)
		require.True(t, ok, rawRes)

		targetedUsers := lo.Filter(*res, func(item apimodels.User, _ int) bool {
			return lo.Contains(
				lo.Map(fixtures, func(item *dao.CredentialsEntity, _ int) apimodels.UserID {
					return apimodels.UserID(item.ID)
				}),
				item.GetID(),
			)
		})

		require.Len(t, targetedUsers, len(fixtures))

		for _, user := range fixtures {
			require.Contains(t, targetedUsers, apimodels.User{
				ID:        apimodels.UserID(user.ID),
				Email:     apimodels.Email(user.Email),
				Role:      apimodels.CredentialsRole(user.Role),
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
			})
		}
	}

	t.Log("FilterByRole")
	{
		fixtures = []*dao.CredentialsEntity{fixtures[0], fixtures[2]}

		rawRes, err := client.ListUsers(t.Context(), apimodels.ListUsersParams{
			Roles: []apimodels.CredentialsRole{apimodels.CredentialsRoleUser},
		})
		require.NoError(t, err)

		res, ok := rawRes.(*apimodels.ListUsersOKApplicationJSON)
		require.True(t, ok, rawRes)

		targetedUsers := lo.Filter(*res, func(item apimodels.User, _ int) bool {
			return lo.Contains(
				lo.Map(fixtures, func(item *dao.CredentialsEntity, _ int) apimodels.UserID {
					return apimodels.UserID(item.ID)
				}),
				item.GetID(),
			)
		})

		require.Len(t, targetedUsers, len(fixtures))

		for _, user := range fixtures {
			require.Contains(t, targetedUsers, apimodels.User{
				ID:        apimodels.UserID(user.ID),
				Email:     apimodels.Email(user.Email),
				Role:      apimodels.CredentialsRole(user.Role),
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
			})
		}
	}
}
