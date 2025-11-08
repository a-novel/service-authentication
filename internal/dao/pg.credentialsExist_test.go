package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/dao"
)

func TestCredentialsExist(t *testing.T) {
	testCases := []struct {
		name string

		fixtures []*dao.Credentials

		request *dao.CredentialsExistRequest

		expect    bool
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.Credentials{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					Role:      "user",
				},
			},

			request: &dao.CredentialsExistRequest{
				Email: "user@provider.com",
			},

			expect: true,
		},
		{
			name: "NotFound",

			request: &dao.CredentialsExistRequest{
				Email: "user@provider.com",
			},

			expect: false,
		},
	}

	repository := dao.NewCredentialsExist()

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

				key, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, key)
			})
		})
	}
}
