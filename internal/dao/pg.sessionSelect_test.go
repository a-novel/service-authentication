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

func TestPgSessionSelect(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hourAgo := now.Add(-time.Hour)
	dayLater := now.Add(24 * time.Hour)

	live := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1"), UserID: sessionOwnerID, UserAgent: "Firefox",
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

	fixtures := sessionFixtures{
		sessions: []*dao.Session{
			live,
			revoked,
			ended,
			{
				ID: uuid.MustParse("00000000-0000-0000-0000-0000000000b1"), UserID: sessionOtherID,
				CreatedAt: hourAgo, LastActiveAt: now, ExpiresAt: dayLater,
			},
		},
		revocations: []*dao.RefreshTokenRevocation{
			{JTI: revoked.ID.String(), ExpiresAt: dayLater, CreatedAt: now},
		},
	}

	testCases := []struct {
		name string

		request *dao.SessionSelectRequest

		expect    *dao.Session
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.SessionSelectRequest{ID: live.ID, UserID: sessionOwnerID},

			expect: live,
		},
		{
			name: "Success/Revoked",

			request: &dao.SessionSelectRequest{ID: revoked.ID, UserID: sessionOwnerID},

			expect: changed(*revoked, func(session *dao.Session) { session.LoggedOut = true }),
		},
		{
			name: "Success/Ended",

			request: &dao.SessionSelectRequest{ID: ended.ID, UserID: sessionOwnerID},

			expect: changed(*ended, func(session *dao.Session) { session.LoggedOut = true }),
		},
		{
			// Another account's session answers as an unknown one.
			name: "Error/OtherAccount",

			request: &dao.SessionSelectRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-0000000000b1"), UserID: sessionOwnerID,
			},

			expectErr: dao.ErrSessionSelectNotFound,
		},
		{
			name: "Error/Unknown",

			request: &dao.SessionSelectRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-0000000000ff"), UserID: sessionOwnerID,
			},

			expectErr: dao.ErrSessionSelectNotFound,
		},
	}

	daoSelect := dao.NewPgSessionSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				fixtures.insert(ctx, t)

				session, err := daoSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, session)
			})
		})
	}
}
