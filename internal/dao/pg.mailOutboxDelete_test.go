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

func TestPgMailOutboxDelete(t *testing.T) {
	t.Parallel()

	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	dayLater := time.Now().Add(24 * time.Hour).UTC().Round(time.Second)

	entry := &dao.MailOutboxEntry{
		ShortCodeID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Usage:       "test-usage",
		Target:      "test-target",
		Recipient:   "user@provider.com",
		Lang:        "fr",
		Attempts:    1,
		DueAt:       hourAgo,
		ExpiresAt:   dayLater,
	}

	testCases := []struct {
		name string

		request *dao.MailOutboxDeleteRequest

		expect     *dao.MailOutboxEntry
		expectLeft []*dao.MailOutboxEntry
		expectErr  error
	}{
		{
			name: "Success",

			request: &dao.MailOutboxDeleteRequest{ShortCodeID: entry.ShortCodeID},

			expect:     entry,
			expectLeft: []*dao.MailOutboxEntry{},
		},
		{
			name: "Error/NotFound",

			request: &dao.MailOutboxDeleteRequest{ShortCodeID: uuid.MustParse("00000000-0000-0000-0000-000000000002")},

			expectLeft: []*dao.MailOutboxEntry{entry},
			expectErr:  dao.ErrMailOutboxDeleteNotFound,
		},
	}

	deleteDao := dao.NewPgMailOutboxDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertMailOutboxFixtures(ctx, t, entry)

				deleted, err := deleteDao.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, deleted)

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				left := make([]*dao.MailOutboxEntry, 0)

				err = db.NewSelect().Model(&left).Scan(ctx)
				require.NoError(t, err)
				require.Equal(t, testCase.expectLeft, left)
			})
		})
	}
}
