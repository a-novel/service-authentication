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

func TestPgRefreshTokenRevocationPurge(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hourAgo := now.Add(-time.Hour)
	hourLater := now.Add(time.Hour)

	testCases := []struct {
		name string

		fixtures []*dao.RefreshTokenRevocation

		request *dao.RefreshTokenRevocationPurgeRequest

		expect *dao.RefreshTokenRevocationPurgeResult
		// expectRemaining lists the JTIs left after the call.
		expectRemaining []string
	}{
		{
			name: "Success",

			// Two expired revocations, the older first, and one still live.
			fixtures: []*dao.RefreshTokenRevocation{
				{JTI: "refresh-token-1", ExpiresAt: hourAgo.Add(-time.Hour), CreatedAt: hourAgo.Add(-2 * time.Hour)},
				{JTI: "refresh-token-2", ExpiresAt: hourAgo, CreatedAt: hourAgo.Add(-time.Hour)},
				{JTI: "refresh-token-3", ExpiresAt: hourLater, CreatedAt: hourAgo},
			},

			request: &dao.RefreshTokenRevocationPurgeRequest{Limit: 10},

			expect:          &dao.RefreshTokenRevocationPurgeResult{Purged: 2},
			expectRemaining: []string{"refresh-token-3"},
		},
		{
			name: "Success/Limit",

			// Two expired revocations, the older first, and one still live.
			fixtures: []*dao.RefreshTokenRevocation{
				{JTI: "refresh-token-1", ExpiresAt: hourAgo.Add(-time.Hour), CreatedAt: hourAgo.Add(-2 * time.Hour)},
				{JTI: "refresh-token-2", ExpiresAt: hourAgo, CreatedAt: hourAgo.Add(-time.Hour)},
				{JTI: "refresh-token-3", ExpiresAt: hourLater, CreatedAt: hourAgo},
			},

			request: &dao.RefreshTokenRevocationPurgeRequest{Limit: 1},

			expect:          &dao.RefreshTokenRevocationPurgeResult{Purged: 1},
			expectRemaining: []string{"refresh-token-2", "refresh-token-3"},
		},
		{
			name: "Success/Empty",

			request: &dao.RefreshTokenRevocationPurgeRequest{Limit: 10},

			expect:          &dao.RefreshTokenRevocationPurgeResult{Purged: 0},
			expectRemaining: []string{},
		},
	}

	daoPurge := dao.NewPgRefreshTokenRevocationPurge()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				res, err := daoPurge.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, res)

				remaining := make([]string, 0)

				err = db.NewSelect().Model((*dao.RefreshTokenRevocation)(nil)).Column("jti").Order("jti").Scan(ctx, &remaining)
				require.NoError(t, err)
				require.Equal(t, testCase.expectRemaining, remaining)
			})
		})
	}
}
