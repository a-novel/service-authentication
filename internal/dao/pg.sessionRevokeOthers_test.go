package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgSessionRevokeOthers(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hourAgo := now.Add(-time.Hour)
	dayLater := now.Add(24 * time.Hour)
	leeway := 5 * time.Minute

	current := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	other := uuid.MustParse("00000000-0000-0000-0000-0000000000a2")
	alreadyRevoked := uuid.MustParse("00000000-0000-0000-0000-0000000000a3")

	testCases := []struct {
		name string

		fixtures sessionFixtures

		request *dao.SessionRevokeOthersRequest

		// expect lists every revocation after the call, by JTI.
		expect []*dao.RefreshTokenRevocation
	}{
		{
			// The current session, an ended one and another account's stay as they were. A session
			// already signed out keeps its first revocation.
			name: "Success",

			fixtures: sessionFixtures{
				sessions: []*dao.Session{
					{ID: current, UserID: sessionOwnerID, CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater},
					{ID: other, UserID: sessionOwnerID, CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater},
					{ID: alreadyRevoked, UserID: sessionOwnerID, CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater},
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a4"), UserID: sessionOwnerID,
						CreatedAt: hourAgo, LastActiveAt: hourAgo, ExpiresAt: hourAgo,
					},
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-0000000000b1"), UserID: sessionOtherID,
						CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater,
					},
				},
				revocations: []*dao.RefreshTokenRevocation{
					{JTI: alreadyRevoked.String(), ExpiresAt: dayLater, CreatedAt: hourAgo},
				},
			},

			request: &dao.SessionRevokeOthersRequest{UserID: sessionOwnerID, KeepID: current, Leeway: leeway, Now: now},

			expect: []*dao.RefreshTokenRevocation{
				{JTI: other.String(), ExpiresAt: dayLater.Add(leeway), CreatedAt: now},
				{JTI: alreadyRevoked.String(), ExpiresAt: dayLater, CreatedAt: hourAgo},
			},
		},
		{
			name: "Success/OnlyCurrent",

			fixtures: sessionFixtures{
				sessions: []*dao.Session{
					{ID: current, UserID: sessionOwnerID, CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater},
				},
			},

			request: &dao.SessionRevokeOthersRequest{UserID: sessionOwnerID, KeepID: current, Leeway: leeway, Now: now},

			expect: []*dao.RefreshTokenRevocation{},
		},
	}

	daoRevokeOthers := dao.NewPgSessionRevokeOthers()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				testCase.fixtures.insert(ctx, t)

				require.NoError(t, daoRevokeOthers.Exec(ctx, testCase.request))

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				revocations := make([]*dao.RefreshTokenRevocation, 0)

				require.NoError(t, db.NewSelect().Model(&revocations).Order("jti").Scan(ctx))
				require.Equal(t, testCase.expect, revocations)
			})
		})
	}
}
