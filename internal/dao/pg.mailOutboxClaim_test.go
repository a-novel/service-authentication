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

func TestPgMailOutboxClaim(t *testing.T) {
	t.Parallel()

	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	now := time.Now().UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)
	dayLater := time.Now().Add(24 * time.Hour).UTC().Round(time.Second)

	shortCodeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	testCases := []struct {
		name string

		dueAt time.Time

		request *dao.MailOutboxClaimRequest

		expect    *dao.MailOutboxEntry
		expectErr error
	}{
		{
			name: "Success",

			dueAt: hourAgo,

			request: &dao.MailOutboxClaimRequest{ShortCodeID: shortCodeID, Now: now, LeaseUntil: hourLater},

			expect: &dao.MailOutboxEntry{
				ShortCodeID: shortCodeID,
				Usage:       "test-usage",
				Target:      "test-target",
				Recipient:   "user@provider.com",
				Lang:        "fr",
				Attempts:    2,
				DueAt:       hourLater,
				ExpiresAt:   dayLater,
			},
		},
		{
			// A claimed entry is due only once its lease ends, so a second claim finds nothing.
			name: "Error/NotDue",

			dueAt: hourLater,

			request: &dao.MailOutboxClaimRequest{ShortCodeID: shortCodeID, Now: now, LeaseUntil: hourLater},

			expectErr: dao.ErrMailOutboxClaimNotFound,
		},
		{
			name: "Error/NotFound",

			dueAt: hourAgo,

			request: &dao.MailOutboxClaimRequest{
				ShortCodeID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:         now,
				LeaseUntil:  hourLater,
			},

			expectErr: dao.ErrMailOutboxClaimNotFound,
		},
	}

	claim := dao.NewPgMailOutboxClaim()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertMailOutboxFixtures(ctx, t, &dao.MailOutboxEntry{
					ShortCodeID: shortCodeID,
					Usage:       "test-usage",
					Target:      "test-target",
					Recipient:   "user@provider.com",
					Lang:        "fr",
					Attempts:    1,
					DueAt:       testCase.dueAt,
					ExpiresAt:   dayLater,
				})

				entry, err := claim.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, entry)
			})
		})
	}
}

// insertMailOutboxFixtures stores entries and the short code each one references.
func insertMailOutboxFixtures(ctx context.Context, t *testing.T, entries ...*dao.MailOutboxEntry) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	shortCodes := make([]*dao.ShortCode, 0, len(entries))
	for _, entry := range entries {
		shortCodes = append(shortCodes, &dao.ShortCode{
			ID:        entry.ShortCodeID,
			Code:      "test-code",
			Usage:     entry.Usage,
			Target:    entry.Target + "-" + entry.ShortCodeID.String(),
			CreatedAt: entry.DueAt,
			ExpiresAt: entry.ExpiresAt,
		})
	}

	_, err = db.NewInsert().Model(&shortCodes).Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&entries).Exec(ctx)
	require.NoError(t, err)
}
