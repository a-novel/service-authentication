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

func TestPgSessionPurge(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	day := 24 * time.Hour
	cutoff := now.Add(-30 * day)

	// newSession returns a session of the owner that ends at end, last active the day before.
	newSession := func(id string, end time.Time) *dao.Session {
		return &dao.Session{
			ID: uuid.MustParse(id), UserID: sessionOwnerID,
			CreatedAt: end.Add(-7 * day), LastActiveAt: end.Add(-day), ExpiresAt: end,
		}
	}

	active := newSession("00000000-0000-0000-0000-0000000000a1", now.Add(day))
	// Ended for less than the cutoff, though inactive for longer.
	recent := newSession("00000000-0000-0000-0000-0000000000a2", now.Add(-29*day))
	old := newSession("00000000-0000-0000-0000-0000000000a3", now.Add(-31*day))
	oldest := newSession("00000000-0000-0000-0000-0000000000a4", now.Add(-40*day))

	fixtures := sessionFixtures{sessions: []*dao.Session{active, recent, old, oldest}}

	testCases := []struct {
		name string

		request *dao.SessionPurgeRequest

		expect     *dao.SessionPurgeResult
		expectLeft []*dao.Session
	}{
		{
			// The rows that ended before the cutoff go, and the newer ones stay.
			name: "Success",

			request: &dao.SessionPurgeRequest{EndedBefore: cutoff, Limit: 10},

			expect:     &dao.SessionPurgeResult{Purged: 2},
			expectLeft: []*dao.Session{active, recent},
		},
		{
			// The row that ended first goes first.
			name: "Success/Limit",

			request: &dao.SessionPurgeRequest{EndedBefore: cutoff, Limit: 1},

			expect:     &dao.SessionPurgeResult{Purged: 1},
			expectLeft: []*dao.Session{active, recent, old},
		},
	}

	daoPurge := dao.NewPgSessionPurge()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				fixtures.insert(ctx, t)

				result, err := daoPurge.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
				require.Equal(t, testCase.expectLeft, selectSessions(ctx, t))
			})
		})
	}
}
