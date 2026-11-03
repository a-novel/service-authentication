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

func TestPgSessionClearLocations(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	dayLater := now.Add(24 * time.Hour)

	newSession := func(id string, userID uuid.UUID) *dao.Session {
		return &dao.Session{
			ID: uuid.MustParse(id), UserID: userID, UserAgent: "Firefox",
			IPAddress: "192.0.2.1", Region: "FR", City: "Paris",
			CreatedAt: now, LastActiveAt: now, ExpiresAt: dayLater,
		}
	}

	// bare is a session of the owner that recorded no location.
	bare := &dao.Session{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a3"), UserID: sessionOwnerID, UserAgent: "Firefox",
		CreatedAt: now, LastActiveAt: now, ExpiresAt: dayLater,
	}

	testCases := []struct {
		name string

		fixtures sessionFixtures

		request *dao.SessionClearLocationsRequest

		expect []*dao.Session
	}{
		{
			// Every session of the account loses its location, and keeps the rest. Another account's
			// sessions keep theirs.
			name: "Success",

			fixtures: sessionFixtures{
				sessions: []*dao.Session{
					newSession("00000000-0000-0000-0000-0000000000a1", sessionOwnerID),
					newSession("00000000-0000-0000-0000-0000000000a2", sessionOwnerID),
					bare,
					newSession("00000000-0000-0000-0000-0000000000b1", sessionOtherID),
				},
			},

			request: &dao.SessionClearLocationsRequest{UserID: sessionOwnerID},

			expect: []*dao.Session{
				{
					ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1"), UserID: sessionOwnerID, UserAgent: "Firefox",
					CreatedAt: now, LastActiveAt: now, ExpiresAt: dayLater,
				},
				{
					ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a2"), UserID: sessionOwnerID, UserAgent: "Firefox",
					CreatedAt: now, LastActiveAt: now, ExpiresAt: dayLater,
				},
				bare,
				newSession("00000000-0000-0000-0000-0000000000b1", sessionOtherID),
			},
		},
	}

	daoClear := dao.NewPgSessionClearLocations()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				testCase.fixtures.insert(ctx, t)

				// A row without a location keeps its version, so a repeated withdrawal rewrites nothing.
				bareVersion := sessionVersion(ctx, t, bare.ID)

				require.NoError(t, daoClear.Exec(ctx, testCase.request))
				require.Equal(t, testCase.expect, selectSessions(ctx, t))
				require.Equal(t, bareVersion, sessionVersion(ctx, t, bare.ID))
			})
		})
	}
}
