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

func TestPgSessionTrim(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hour := time.Hour
	dayLater := now.Add(24 * hour)

	// newSession returns a session of userID last active lastActive ago, ending at end.
	newSession := func(id string, userID uuid.UUID, lastActive time.Duration, end time.Time) *dao.Session {
		return &dao.Session{
			ID: uuid.MustParse(id), UserID: userID,
			CreatedAt: now.Add(-48 * hour), LastActiveAt: now.Add(-lastActive), ExpiresAt: end,
		}
	}

	// An active session older than every logged-out one stays.
	active := newSession("00000000-0000-0000-0000-0000000000a1", sessionOwnerID, 40*hour, dayLater)
	revokedNewest := newSession("00000000-0000-0000-0000-0000000000a2", sessionOwnerID, hour, dayLater)
	endedMiddle := newSession("00000000-0000-0000-0000-0000000000a3", sessionOwnerID, 2*hour, now.Add(-hour))
	endedOldest := newSession("00000000-0000-0000-0000-0000000000a4", sessionOwnerID, 3*hour, now.Add(-hour))
	foreign := newSession("00000000-0000-0000-0000-0000000000b1", sessionOtherID, 4*hour, now.Add(-hour))

	fixtures := sessionFixtures{
		sessions: []*dao.Session{active, revokedNewest, endedMiddle, endedOldest, foreign},
		revocations: []*dao.RefreshTokenRevocation{
			{JTI: revokedNewest.ID.String(), ExpiresAt: dayLater, CreatedAt: now},
		},
	}

	testCases := []struct {
		name string

		request *dao.SessionTrimRequest

		expectLeft []*dao.Session
	}{
		{
			// The account keeps its logged-out session active last, revoked or ended, and every active one.
			// Another account's sessions stay.
			name: "Success",

			request: &dao.SessionTrimRequest{UserID: sessionOwnerID, Keep: 1, Limit: 10},

			expectLeft: []*dao.Session{active, revokedNewest, foreign},
		},
		{
			// The deletion stops at the limit, right after the sessions kept.
			name: "Success/Limit",

			request: &dao.SessionTrimRequest{UserID: sessionOwnerID, Keep: 1, Limit: 1},

			expectLeft: []*dao.Session{active, revokedNewest, endedOldest, foreign},
		},
		{
			name: "Success/UnderKeep",

			request: &dao.SessionTrimRequest{UserID: sessionOwnerID, Keep: 3, Limit: 10},

			expectLeft: []*dao.Session{active, revokedNewest, endedMiddle, endedOldest, foreign},
		},
	}

	daoTrim := dao.NewPgSessionTrim()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				fixtures.insert(ctx, t)

				require.NoError(t, daoTrim.Exec(ctx, testCase.request))
				require.Equal(t, testCase.expectLeft, selectSessions(ctx, t))
			})
		})
	}
}
