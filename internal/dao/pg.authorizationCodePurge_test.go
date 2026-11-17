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

func TestPgAuthorizationCodePurge(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)

	// Two expired codes, the older first, and one still live. Each code's client names it.
	fixtures := []*dao.AuthorizationCode{
		newAuthorizationCode("code-1", now.Add(-2*time.Minute)),
		newAuthorizationCode("code-2", now.Add(-time.Minute)),
		// An hour ahead: a loaded test run can take minutes to reach the purge.
		newAuthorizationCode("code-3", now.Add(time.Hour)),
	}

	testCases := []struct {
		name string

		fixtures []*dao.AuthorizationCode

		request *dao.AuthorizationCodePurgeRequest

		expect *dao.AuthorizationCodePurgeResult
		// expectRemaining lists the clients of the codes left after the call.
		expectRemaining []string
	}{
		{
			name: "Success",

			fixtures: fixtures,

			request: &dao.AuthorizationCodePurgeRequest{Limit: 10},

			expect:          &dao.AuthorizationCodePurgeResult{Purged: 2},
			expectRemaining: []string{"code-3"},
		},
		{
			name: "Success/Limit",

			fixtures: fixtures,

			request: &dao.AuthorizationCodePurgeRequest{Limit: 1},

			expect:          &dao.AuthorizationCodePurgeResult{Purged: 1},
			expectRemaining: []string{"code-2", "code-3"},
		},
		{
			name: "Success/Empty",

			request: &dao.AuthorizationCodePurgeRequest{Limit: 10},

			expect:          &dao.AuthorizationCodePurgeResult{Purged: 0},
			expectRemaining: []string{},
		},
	}

	daoPurge := dao.NewPgAuthorizationCodePurge()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertAccounts(ctx, t)
				insertFixtures(ctx, t, testCase.fixtures)

				res, err := daoPurge.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, res)

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				remaining := make([]string, 0)

				err = db.NewSelect().
					Model((*dao.AuthorizationCode)(nil)).
					Column("client_id").
					Order("client_id").
					Scan(ctx, &remaining)
				require.NoError(t, err)
				require.Equal(t, testCase.expectRemaining, remaining)
			})
		})
	}
}
