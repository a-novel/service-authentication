package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestShortCodeDelete(t *testing.T) {
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	now := time.Now().UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

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
				DeletedComment: lo.ToPtr("foo"),
			},
		},
		{
			name: "NotFound",

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

			expectErr: dao.ErrShortCodeDeleteNotFound,
		},
		{
			name: "AlreadyExpired",

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

			expectErr: dao.ErrShortCodeDeleteNotFound,
		},
		{
			name: "DeleteMultipleTimes",

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
					DeletedAt:      lo.ToPtr(now.Add(-30 * time.Minute)),
					DeletedComment: lo.ToPtr("bar"),
				},
			},

			expectErr: dao.ErrShortCodeDeleteNotFound,
		},
	}

	repository := dao.NewShortCodeDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				key, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, key)
			})
		})
	}
}
