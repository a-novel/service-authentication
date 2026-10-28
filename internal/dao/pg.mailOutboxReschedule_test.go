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

func TestPgMailOutboxReschedule(t *testing.T) {
	t.Parallel()

	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)
	dayLater := time.Now().Add(24 * time.Hour).UTC().Round(time.Second)

	shortCodeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	testCases := []struct {
		name string

		request *dao.MailOutboxRescheduleRequest

		expect    *dao.MailOutboxEntry
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.MailOutboxRescheduleRequest{ShortCodeID: shortCodeID, DueAt: hourLater},

			expect: &dao.MailOutboxEntry{
				ShortCodeID: shortCodeID,
				Usage:       "test-usage",
				Target:      "test-target",
				Recipient:   "user@provider.com",
				Lang:        "fr",
				Attempts:    1,
				DueAt:       hourLater,
				ExpiresAt:   dayLater,
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.MailOutboxRescheduleRequest{
				ShortCodeID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				DueAt:       hourLater,
			},

			expectErr: dao.ErrMailOutboxRescheduleNotFound,
		},
	}

	reschedule := dao.NewPgMailOutboxReschedule()

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
					DueAt:       hourAgo,
					ExpiresAt:   dayLater,
				})

				entry, err := reschedule.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, entry)
			})
		})
	}
}
