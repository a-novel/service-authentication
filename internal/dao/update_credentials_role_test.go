package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/models"
	"github.com/a-novel/service-authentication/models/config"
)

func TestUpdateCredentialsRole(t *testing.T) {
	testCases := []struct {
		name string

		fixtures []*dao.CredentialsEntity

		userID     uuid.UUID
		updateData dao.UpdateCredentialsRoleData

		expect    *dao.CredentialsEntity
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.CredentialsEntity{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					Role:      models.CredentialsRoleUser,
				},
			},

			userID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

			updateData: dao.UpdateCredentialsRoleData{
				Role: models.CredentialsRoleAdmin,
				Now:  time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.CredentialsEntity{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Email:     "user@provider.com",
				Password:  "password-2-hashed",
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				Role:      models.CredentialsRoleAdmin,
			},
		},
		{
			name: "Error/NotFound",

			userID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

			updateData: dao.UpdateCredentialsRoleData{
				Role: models.CredentialsRoleAdmin,
				Now:  time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expectErr: dao.ErrCredentialsNotFound,
		},
	}

	repository := dao.NewUpdateCredentialsRoleRepository()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				credentials, err := repository.UpdateCredentialsRole(ctx, testCase.userID, testCase.updateData)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, credentials)
			})
		})
	}
}
