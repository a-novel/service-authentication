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

func TestPgSessionDeleteLoggedOut(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hourAgo := now.Add(-time.Hour)
	dayLater := now.Add(24 * time.Hour)

	active := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1"), UserID: sessionOwnerID,
		CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater,
	}
	revoked := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a2"), UserID: sessionOwnerID,
		CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater,
	}
	ended := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a3"), UserID: sessionOwnerID,
		CreatedAt: hourAgo, LastActiveAt: hourAgo, ExpiresAt: hourAgo,
	}
	foreign := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000b1"), UserID: sessionOtherID,
		CreatedAt: hourAgo, LastActiveAt: hourAgo, ExpiresAt: hourAgo,
	}

	revocation := &dao.RefreshTokenRevocation{JTI: revoked.ID.String(), ExpiresAt: dayLater, CreatedAt: now}

	testCases := []struct {
		name string

		fixtures sessionFixtures

		request *dao.SessionDeleteLoggedOutRequest

		expectLeft []*dao.Session
		// expectRevocations lists every revocation after the call.
		expectRevocations []*dao.RefreshTokenRevocation
	}{
		{
			// The active session and another account's logged-out one keep their rows. The revocation
			// stays, so its session stays signed out.
			name: "Success",

			fixtures: sessionFixtures{
				sessions:    []*dao.Session{active, revoked, ended, foreign},
				revocations: []*dao.RefreshTokenRevocation{revocation},
			},

			request: &dao.SessionDeleteLoggedOutRequest{UserID: sessionOwnerID},

			expectLeft:        []*dao.Session{active, foreign},
			expectRevocations: []*dao.RefreshTokenRevocation{revocation},
		},
		{
			name: "Success/NoneLoggedOut",

			fixtures: sessionFixtures{sessions: []*dao.Session{active, foreign}},

			request: &dao.SessionDeleteLoggedOutRequest{UserID: sessionOwnerID},

			expectLeft:        []*dao.Session{active, foreign},
			expectRevocations: []*dao.RefreshTokenRevocation{},
		},
	}

	daoDeleteLoggedOut := dao.NewPgSessionDeleteLoggedOut()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				testCase.fixtures.insert(ctx, t)

				require.NoError(t, daoDeleteLoggedOut.Exec(ctx, testCase.request))
				require.Equal(t, testCase.expectLeft, selectSessions(ctx, t))

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				revocations := make([]*dao.RefreshTokenRevocation, 0)

				require.NoError(t, db.NewSelect().Model(&revocations).Scan(ctx))
				require.Equal(t, testCase.expectRevocations, revocations)
			})
		})
	}
}
