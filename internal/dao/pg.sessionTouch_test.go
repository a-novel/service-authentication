package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgSessionTouch(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hourAgo := now.Add(-time.Hour)
	dayLater := now.Add(24 * time.Hour)
	sessionID := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")

	testCases := []struct {
		name string

		fixtures sessionFixtures

		request *dao.SessionTouchRequest

		expect []*dao.Session
	}{
		{
			// A session an older release opened gets a row with no device.
			name: "Success/Open",

			request: &dao.SessionTouchRequest{ID: sessionID, UserID: sessionOwnerID, Now: now, ExpiresAt: dayLater},

			expect: []*dao.Session{
				{ID: sessionID, UserID: sessionOwnerID, CreatedAt: now, LastActiveAt: now, ExpiresAt: dayLater},
			},
		},
		{
			// A known session keeps everything but its last activity.
			name: "Success/Touch",

			fixtures: sessionFixtures{
				sessions: []*dao.Session{
					{
						ID: sessionID, UserID: sessionOwnerID, UserAgent: "Firefox",
						IPAddress: "192.0.2.1", Region: "FR", City: "Paris",
						CreatedAt: hourAgo, LastActiveAt: hourAgo, ExpiresAt: dayLater,
					},
				},
			},

			request: &dao.SessionTouchRequest{
				ID: sessionID, UserID: sessionOwnerID, Now: now, ExpiresAt: dayLater.Add(time.Hour),
			},

			expect: []*dao.Session{
				{
					ID: sessionID, UserID: sessionOwnerID, UserAgent: "Firefox",
					IPAddress: "192.0.2.1", Region: "FR", City: "Paris",
					CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater,
				},
			},
		},
	}

	daoTouch := dao.NewPgSessionTouch()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				testCase.fixtures.insert(ctx, t)

				require.NoError(t, daoTouch.Exec(ctx, testCase.request))
				require.Equal(t, testCase.expect, selectSessions(ctx, t))
			})
		})
	}
}
