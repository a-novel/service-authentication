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

func TestPgSessionDelete(t *testing.T) {
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

	fixtures := sessionFixtures{
		sessions:    []*dao.Session{active, revoked, ended, foreign},
		revocations: []*dao.RefreshTokenRevocation{revocation},
	}

	testCases := []struct {
		name string

		request *dao.SessionDeleteRequest

		expectErr  error
		expectLeft []*dao.Session
	}{
		{
			// The revocation stays, so the session stays signed out.
			name: "Success/Revoked",

			request: &dao.SessionDeleteRequest{ID: revoked.ID, UserID: sessionOwnerID},

			expectLeft: []*dao.Session{active, ended, foreign},
		},
		{
			name: "Success/Ended",

			request: &dao.SessionDeleteRequest{ID: ended.ID, UserID: sessionOwnerID},

			expectLeft: []*dao.Session{active, revoked, foreign},
		},
		{
			// An active session keeps its row.
			name: "Error/Active",

			request: &dao.SessionDeleteRequest{ID: active.ID, UserID: sessionOwnerID},

			expectErr:  dao.ErrSessionDeleteNotFound,
			expectLeft: []*dao.Session{active, revoked, ended, foreign},
		},
		{
			name: "Error/OtherAccount",

			request: &dao.SessionDeleteRequest{ID: foreign.ID, UserID: sessionOwnerID},

			expectErr:  dao.ErrSessionDeleteNotFound,
			expectLeft: []*dao.Session{active, revoked, ended, foreign},
		},
	}

	daoDelete := dao.NewPgSessionDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				fixtures.insert(ctx, t)

				err := daoDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expectLeft, selectSessions(ctx, t))

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				revocations := make([]*dao.RefreshTokenRevocation, 0)

				require.NoError(t, db.NewSelect().Model(&revocations).Scan(ctx))
				require.Equal(t, []*dao.RefreshTokenRevocation{revocation}, revocations)
			})
		})
	}
}
