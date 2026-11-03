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

	// newSession returns a session of the owner last active at lastActive, which ended a week later.
	newSession := func(id string, lastActive time.Time) *dao.Session {
		return &dao.Session{
			ID: uuid.MustParse(id), UserID: sessionOwnerID,
			CreatedAt: lastActive, LastActiveAt: lastActive, ExpiresAt: lastActive.Add(7 * day),
		}
	}

	active := newSession("00000000-0000-0000-0000-0000000000a1", now)
	// Logged out, inactive for less than the cutoff.
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
			// The rows inactive since before the cutoff go, and the newer ones stay.
			name: "Success",

			request: &dao.SessionPurgeRequest{ActiveBefore: cutoff, Limit: 10},

			expect:     &dao.SessionPurgeResult{Purged: 2},
			expectLeft: []*dao.Session{active, recent},
		},
		{
			// The least recent row goes first.
			name: "Success/Limit",

			request: &dao.SessionPurgeRequest{ActiveBefore: cutoff, Limit: 1},

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
