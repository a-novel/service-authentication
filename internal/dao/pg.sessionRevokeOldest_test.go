package dao_test

import (
	"context"
	"fmt"
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

func TestPgSessionRevokeOldest(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	dayLater := now.Add(24 * time.Hour)
	leeway := 5 * time.Minute

	// liveSessions returns count live sessions of the owner, the first active last, one hour apart.
	liveSessions := func(count int) []*dao.Session {
		sessions := make([]*dao.Session, count)
		for i := range sessions {
			lastActive := now.Add(-time.Duration(i) * time.Hour)
			sessions[i] = &dao.Session{
				ID:     uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)),
				UserID: sessionOwnerID, CreatedAt: lastActive, LastActiveAt: lastActive, ExpiresAt: dayLater,
			}
		}

		return sessions
	}

	// Logged-out sessions and another account's, all older than every live one, never count.
	weekAgo := now.Add(-7 * 24 * time.Hour)
	ended := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000e1"), UserID: sessionOwnerID,
		CreatedAt: weekAgo, LastActiveAt: weekAgo, ExpiresAt: now.Add(-time.Hour),
	}
	revoked := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000e2"), UserID: sessionOwnerID,
		CreatedAt: weekAgo, LastActiveAt: weekAgo, ExpiresAt: dayLater,
	}
	foreign := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000f1"), UserID: sessionOtherID,
		CreatedAt: weekAgo, LastActiveAt: weekAgo, ExpiresAt: dayLater,
	}
	revocation := &dao.RefreshTokenRevocation{JTI: revoked.ID.String(), ExpiresAt: dayLater, CreatedAt: weekAgo}

	// newRevocation returns the revocation the call records for session.
	newRevocation := func(session *dao.Session) *dao.RefreshTokenRevocation {
		return &dao.RefreshTokenRevocation{JTI: session.ID.String(), ExpiresAt: dayLater.Add(leeway), CreatedAt: now}
	}

	live101 := liveSessions(101)
	live100 := liveSessions(100)

	testCases := []struct {
		name string

		live []*dao.Session

		request *dao.SessionRevokeOldestRequest

		// expect lists every revocation after the call, by JTI.
		expect []*dao.RefreshTokenRevocation
	}{
		{
			// The login that opens the 101st live session signs out the least recent one.
			name: "Success/OverCap",

			live: live101,

			request: &dao.SessionRevokeOldestRequest{
				UserID: sessionOwnerID, Keep: 100, Limit: 1000, Leeway: leeway, Now: now,
			},

			expect: []*dao.RefreshTokenRevocation{revocation, newRevocation(live101[100])},
		},
		{
			name: "Success/AtCap",

			live: live100,

			request: &dao.SessionRevokeOldestRequest{
				UserID: sessionOwnerID, Keep: 100, Limit: 1000, Leeway: leeway, Now: now,
			},

			expect: []*dao.RefreshTokenRevocation{revocation},
		},
		{
			// The revocations stop at the limit, the least recent first.
			name: "Success/Limit",

			live: live101,

			request: &dao.SessionRevokeOldestRequest{
				UserID: sessionOwnerID, Keep: 98, Limit: 2, Leeway: leeway, Now: now,
			},

			expect: []*dao.RefreshTokenRevocation{revocation, newRevocation(live101[99]), newRevocation(live101[100])},
		},
	}

	daoRevokeOldest := dao.NewPgSessionRevokeOldest()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				fixtures := sessionFixtures{
					sessions:    append([]*dao.Session{ended, revoked, foreign}, testCase.live...),
					revocations: []*dao.RefreshTokenRevocation{revocation},
				}
				fixtures.insert(ctx, t)

				require.NoError(t, daoRevokeOldest.Exec(ctx, testCase.request))

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				revocations := make([]*dao.RefreshTokenRevocation, 0)

				require.NoError(t, db.NewSelect().Model(&revocations).Order("jti").Scan(ctx))
				require.Equal(t, testCase.expect, revocations)
			})
		})
	}
}
