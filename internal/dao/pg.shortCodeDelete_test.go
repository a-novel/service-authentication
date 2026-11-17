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

func TestPgShortCodeDelete(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hourAgo := now.Add(-time.Hour)
	hourLater := now.Add(time.Hour)

	testCases := []struct {
		name string

		request  *dao.ShortCodeDeleteRequest
		fixtures []*dao.ShortCode

		expect    *dao.ShortCode
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ShortCodeDeleteRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     "test",
					Target:    "test-target-1",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Usage:     "test",
					Target:    "test-target-2",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
			},

			expect: &dao.ShortCode{
				ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Usage:          "test",
				Target:         "test-target-2",
				CreatedAt:      hourAgo,
				ExpiresAt:      hourLater,
				DeletedAt:      &now,
				DeletedComment: new("foo"),
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.ShortCodeDeleteRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     "test",
					Target:    "test-target-1",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
			},

			expectErr: dao.ErrShortCodeNotFound,
		},
		{
			name: "Error/Expired",

			request: &dao.ShortCodeDeleteRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     "test",
					Target:    "test-target-1",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Usage:     "test",
					Target:    "test-target-2",
					CreatedAt: hourAgo,
					ExpiresAt: hourAgo,
				},
			},

			expectErr: dao.ErrShortCodeNotFound,
		},
		{
			name: "Error/AlreadyDeleted",

			request: &dao.ShortCodeDeleteRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     "test",
					Target:    "test-target-1",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
				{
					ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Usage:          "test",
					Target:         "test-target-2",
					CreatedAt:      hourAgo,
					ExpiresAt:      hourLater,
					DeletedAt:      new(now.Add(-30 * time.Minute)),
					DeletedComment: new("bar"),
				},
			},

			expectErr: dao.ErrShortCodeNotFound,
		},
	}

	daoDelete := dao.NewPgShortCodeDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, testCase.fixtures)

				shortCode, err := daoDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, shortCode)
			})
		})
	}
}
