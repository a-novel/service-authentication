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

func TestPgMailOutboxList(t *testing.T) {
	t.Parallel()

	twoHoursAgo := time.Now().Add(-2 * time.Hour).UTC().Round(time.Second)
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	now := time.Now().UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)
	dayLater := time.Now().Add(24 * time.Hour).UTC().Round(time.Second)

	newEntry := func(id string, dueAt time.Time) *dao.MailOutboxEntry {
		return &dao.MailOutboxEntry{
			ShortCodeID: uuid.MustParse(id),
			Usage:       "test-usage",
			Target:      "test-target",
			Recipient:   "user@provider.com",
			Lang:        "fr",
			DueAt:       dueAt,
			ExpiresAt:   dayLater,
		}
	}

	dueRecently := newEntry("00000000-0000-0000-0000-000000000001", hourAgo)
	dueLongest := newEntry("00000000-0000-0000-0000-000000000002", twoHoursAgo)
	notDue := newEntry("00000000-0000-0000-0000-000000000003", hourLater)

	testCases := []struct {
		name string

		fixtures []*dao.MailOutboxEntry
		request  *dao.MailOutboxListRequest

		expect []*dao.MailOutboxEntry
	}{
		{
			name: "Success",

			fixtures: []*dao.MailOutboxEntry{dueRecently, dueLongest, notDue},
			request:  &dao.MailOutboxListRequest{Now: now, Limit: 10},

			expect: []*dao.MailOutboxEntry{dueLongest, dueRecently},
		},
		{
			name: "Success/Limit",

			fixtures: []*dao.MailOutboxEntry{dueRecently, dueLongest, notDue},
			request:  &dao.MailOutboxListRequest{Now: now, Limit: 1},

			expect: []*dao.MailOutboxEntry{dueLongest},
		},
		{
			name: "Success/Empty",

			fixtures: []*dao.MailOutboxEntry{notDue},
			request:  &dao.MailOutboxListRequest{Now: now, Limit: 10},

			expect: []*dao.MailOutboxEntry{},
		},
	}

	list := dao.NewPgMailOutboxList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertMailOutboxFixtures(ctx, t, testCase.fixtures...)

				entries, err := list.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, entries)
			})
		})
	}
}
