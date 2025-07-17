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
	testutils "github.com/a-novel/service-authentication/pkg/cmd"
)

func TestInsertCredentials(t *testing.T) {
	testCases := []struct {
		name string

		fixtures []*dao.CredentialsEntity

		insertData dao.InsertCredentialsData

		expect    *dao.CredentialsEntity
		expectErr error
	}{
		{
			name: "Success",

			insertData: dao.InsertCredentialsData{
				ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:    "user@provider.com",
				Password: "password-hashed",
				Now:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.CredentialsEntity{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "user@provider.com",
				Password:  "password-hashed",
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				Role:      models.CredentialsRoleUser,
			},
		},
		{
			name: "NoPassword",

			insertData: dao.InsertCredentialsData{
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email: "user@provider.com",
				Now:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.CredentialsEntity{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:     "user@provider.com",
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				Role:      models.CredentialsRoleUser,
			},
		},
		{
			name: "Error/AlreadyExists",

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

			insertData: dao.InsertCredentialsData{
				ID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Email:    "user@provider.com",
				Password: "password-hashed",
				Now:      time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},

			expectErr: dao.ErrCredentialsAlreadyExists,
		},
	}

	repository := dao.NewInsertCredentialsRepository()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			postgres.RunTransactionalTest(t, testutils.PostgresConfigTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				credentials, err := repository.InsertCredentials(ctx, testCase.insertData)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, credentials)
			})
		})
	}
}
