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

func TestPgSessionList(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hourAgo := now.Add(-time.Hour)
	dayLater := now.Add(24 * time.Hour)

	recent := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1"), UserID: sessionOwnerID, UserAgent: "Firefox",
		CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater,
	}
	older := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a2"), UserID: sessionOwnerID, UserAgent: "Chrome",
		IPAddress: "2001:db8::1", Region: "FR", City: "Paris",
		CreatedAt: hourAgo, LastActiveAt: hourAgo, ExpiresAt: dayLater,
	}
	ended := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a3"), UserID: sessionOwnerID,
		CreatedAt: hourAgo, LastActiveAt: hourAgo, ExpiresAt: hourAgo,
	}
	revoked := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a4"), UserID: sessionOwnerID,
		CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater,
	}

	fixtures := sessionFixtures{
		sessions: []*dao.Session{
			older,
			recent,
			ended,
			revoked,
			{
				ID: uuid.MustParse("00000000-0000-0000-0000-0000000000b1"), UserID: sessionOtherID,
				CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater,
			},
		},
		revocations: []*dao.RefreshTokenRevocation{
			{JTI: revoked.ID.String(), ExpiresAt: dayLater, CreatedAt: now},
		},
	}

	loggedOut := func(session *dao.Session) *dao.Session {
		return changed(*session, func(session *dao.Session) { session.LoggedOut = true })
	}

	testCases := []struct {
		name string

		request *dao.SessionListRequest

		expect []*dao.Session
	}{
		{
			// The active sessions come first, then the ended and the signed-out ones, each latest activity
			// first. Another account's sessions stay out.
			name: "Success",

			request: &dao.SessionListRequest{UserID: sessionOwnerID, Limit: 100},

			expect: []*dao.Session{recent, older, loggedOut(revoked), loggedOut(ended)},
		},
		{
			name: "Success/Limit",

			request: &dao.SessionListRequest{UserID: sessionOwnerID, Limit: 1},

			expect: []*dao.Session{recent},
		},
		{
			name: "Success/Empty",

			request: &dao.SessionListRequest{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Limit: 100},

			expect: []*dao.Session{},
		},
	}

	daoList := dao.NewPgSessionList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				fixtures.insert(ctx, t)

				sessions, err := daoList.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, sessions)
			})
		})
	}
}
