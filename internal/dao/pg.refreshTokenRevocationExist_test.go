package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgRefreshTokenRevocationExist(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		fixtures []*dao.RefreshTokenRevocation

		request *dao.RefreshTokenRevocationExistRequest

		expect bool
	}{
		{
			name: "Success/Revoked",

			fixtures: []*dao.RefreshTokenRevocation{
				{
					JTI:       "refresh-token-1",
					ExpiresAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.RefreshTokenRevocationExistRequest{JTIs: []string{"refresh-token-1", "refresh-token-1"}},

			expect: true,
		},
		{
			// A token of a session another token opened stops once that session is revoked.
			name: "Success/SessionRevoked",

			fixtures: []*dao.RefreshTokenRevocation{
				{
					JTI:       "session-1",
					ExpiresAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.RefreshTokenRevocationExistRequest{JTIs: []string{"refresh-token-1", "session-1"}},

			expect: true,
		},
		{
			// Another session's revocation leaves this one untouched.
			name: "Success/NotRevoked",

			fixtures: []*dao.RefreshTokenRevocation{
				{
					JTI:       "refresh-token-2",
					ExpiresAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},

			request: &dao.RefreshTokenRevocationExistRequest{JTIs: []string{"refresh-token-1", "session-1"}},

			expect: false,
		},
	}

	dao := dao.NewPgRefreshTokenRevocationExist()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
				require.NoError(t, err)

				revoked, err := dao.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, revoked)
			})
		})
	}
}
