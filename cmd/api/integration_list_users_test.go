package main

import (
	"crypto/rand"
	"github.com/a-novel/service-authentication/internal/lib"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/api/apiclient/testapiclient"
	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
)

func TestListUsers(t *testing.T) {
	client, securityClient, err := testapiclient.GetServerClient()
	require.NoError(t, err)

	// YOLO
	ctx, err := lib.NewPostgresContext(t.Context(), os.Getenv("DSN"))
	require.NoError(t, err)

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

	tx, err := lib.PostgresContext(ctx)
	require.NoError(t, err)

	_, err = tx.NewInsert().Model(&fixtures).Exec(ctx)
	require.NoError(t, err)

	token := authAnon(t, client)
	securityClient.SetToken(token)

	// Verify the fixtures are present in the list of returned users.
	t.Log("ListUsers")
	{
		rawRes, err := client.ListUsers(t.Context(), codegen.ListUsersParams{})
		require.NoError(t, err)

		res, ok := rawRes.(*codegen.ListUsersOKApplicationJSON)
		require.True(t, ok)

		targetedUsers := lo.Filter(*res, func(item codegen.User, _ int) bool {
			return lo.Contains(
				lo.Map(fixtures, func(item *dao.CredentialsEntity, _ int) codegen.UserID {
					return codegen.UserID(item.ID)
				}),
				item.GetID(),
			)
		})

		require.Len(t, targetedUsers, len(fixtures))

		for _, user := range targetedUsers {
			require.Contains(t, fixtures, &dao.CredentialsEntity{
				ID:        uuid.UUID(user.GetID()),
				Email:     string(user.GetEmail()),
				Role:      models.CredentialsRole(user.GetRole()),
				CreatedAt: user.GetCreatedAt(),
				UpdatedAt: user.GetUpdatedAt(),
			})
		}
	}

	t.Log("FilterByRole")
	{
		fixtures = []*dao.CredentialsEntity{fixtures[0], fixtures[2]}

		rawRes, err := client.ListUsers(t.Context(), codegen.ListUsersParams{
			Roles: []codegen.CredentialsRole{codegen.CredentialsRoleUser},
		})
		require.NoError(t, err)

		res, ok := rawRes.(*codegen.ListUsersOKApplicationJSON)
		require.True(t, ok)

		targetedUsers := lo.Filter(*res, func(item codegen.User, _ int) bool {
			return lo.Contains(
				lo.Map(fixtures, func(item *dao.CredentialsEntity, _ int) codegen.UserID {
					return codegen.UserID(item.ID)
				}),
				item.GetID(),
			)
		})

		require.Len(t, targetedUsers, len(fixtures))

		for _, user := range targetedUsers {
			require.Contains(t, fixtures, &dao.CredentialsEntity{
				ID:        uuid.UUID(user.GetID()),
				Email:     string(user.GetEmail()),
				Role:      models.CredentialsRole(user.GetRole()),
				CreatedAt: user.GetCreatedAt(),
				UpdatedAt: user.GetUpdatedAt(),
			})
		}
	}
}
