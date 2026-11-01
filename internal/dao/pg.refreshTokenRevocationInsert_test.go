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

func TestPgRefreshTokenRevocationInsert(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hourAgo := now.Add(-time.Hour)
	hourLater := now.Add(time.Hour)

	testCases := []struct {
		name string

		fixtures []*dao.RefreshTokenRevocation

		request *dao.RefreshTokenRevocationInsertRequest

		// expect lists the rows not yet expired after the call, by JTI.
		expect        []*dao.RefreshTokenRevocation
		expectExpired int64
	}{
		{
			name: "Success",

			request: &dao.RefreshTokenRevocationInsertRequest{
				JTIs:      []string{"refresh-token-1", "session-1"},
				ExpiresAt: hourLater,
				Now:       now,
			},

			expect: []*dao.RefreshTokenRevocation{
				{JTI: "refresh-token-1", ExpiresAt: hourLater, CreatedAt: now},
				{JTI: "session-1", ExpiresAt: hourLater, CreatedAt: now},
			},
		},
		{
			// A token that opened its session is the session, so its JTI comes twice.
			name: "Success/SameJTI",

			request: &dao.RefreshTokenRevocationInsertRequest{
				JTIs:      []string{"refresh-token-1", "refresh-token-1"},
				ExpiresAt: hourLater,
				Now:       now,
			},

			expect: []*dao.RefreshTokenRevocation{
				{JTI: "refresh-token-1", ExpiresAt: hourLater, CreatedAt: now},
			},
		},
		{
			// Signing out of a second token of a session keeps the session's first row.
			name: "Success/AlreadyRevoked",

			fixtures: []*dao.RefreshTokenRevocation{
				{JTI: "session-1", ExpiresAt: hourLater, CreatedAt: hourAgo},
			},

			request: &dao.RefreshTokenRevocationInsertRequest{
				JTIs:      []string{"refresh-token-1", "session-1"},
				ExpiresAt: hourLater.Add(time.Hour),
				Now:       now,
			},

			expect: []*dao.RefreshTokenRevocation{
				{JTI: "refresh-token-1", ExpiresAt: hourLater.Add(time.Hour), CreatedAt: now},
				{JTI: "session-1", ExpiresAt: hourLater, CreatedAt: hourAgo},
			},
		},
		{
			name: "Success/ExpiredToken",

			request: &dao.RefreshTokenRevocationInsertRequest{
				JTIs:      []string{"refresh-token-1", "refresh-token-1"},
				ExpiresAt: hourAgo,
				Now:       now,
			},

			expect:        []*dao.RefreshTokenRevocation{},
			expectExpired: 1,
		},
	}

	daoInsert := dao.NewPgRefreshTokenRevocationInsert()

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

				require.NoError(t, daoInsert.Exec(ctx, testCase.request))

				live := make([]*dao.RefreshTokenRevocation, 0)

				err = db.NewSelect().Model(&live).Where("expires_at >= CURRENT_TIMESTAMP").Order("jti").Scan(ctx)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, live)

				expired, err := db.NewSelect().
					Model((*dao.RefreshTokenRevocation)(nil)).
					Where("expires_at < CURRENT_TIMESTAMP").
					Count(ctx)
				require.NoError(t, err)
				require.Equal(t, testCase.expectExpired, expired)
			})
		})
	}
}
