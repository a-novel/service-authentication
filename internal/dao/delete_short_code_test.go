package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/dao"
	testutils "github.com/a-novel/service-authentication/internal/test"
	"github.com/a-novel/service-authentication/models"
)

func TestDeleteShortCode(t *testing.T) {
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	now := time.Now().UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		data     dao.DeleteShortCodeData
		fixtures []*dao.ShortCodeEntity

		expect    *dao.ShortCodeEntity
		expectErr error
	}{
		{
			name: "Success",

			data: dao.DeleteShortCodeData{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.ShortCodeEntity{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     models.ShortCodeUsage("test"),
					Target:    "test-target-1",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Usage:     models.ShortCodeUsage("test"),
					Target:    "test-target-2",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
			},

			expect: &dao.ShortCodeEntity{
				ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Usage:          models.ShortCodeUsage("test"),
				Target:         "test-target-2",
				CreatedAt:      hourAgo,
				ExpiresAt:      hourLater,
				DeletedAt:      &now,
				DeletedComment: lo.ToPtr("foo"),
			},
		},
		{
			name: "NotFound",

			data: dao.DeleteShortCodeData{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.ShortCodeEntity{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     models.ShortCodeUsage("test"),
					Target:    "test-target-1",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
			},

			expectErr: dao.ErrShortCodeNotFound,
		},
		{
			name: "AlreadyExpired",

			data: dao.DeleteShortCodeData{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.ShortCodeEntity{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     models.ShortCodeUsage("test"),
					Target:    "test-target-1",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Usage:     models.ShortCodeUsage("test"),
					Target:    "test-target-2",
					CreatedAt: hourAgo,
					ExpiresAt: hourAgo,
				},
			},

			expectErr: dao.ErrShortCodeNotFound,
		},
		{
			name: "DeleteMultipleTimes",

			data: dao.DeleteShortCodeData{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now:     now,
				Comment: "foo",
			},

			fixtures: []*dao.ShortCodeEntity{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     models.ShortCodeUsage("test"),
					Target:    "test-target-1",
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
				{
					ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Usage:          models.ShortCodeUsage("test"),
					Target:         "test-target-2",
					CreatedAt:      hourAgo,
					ExpiresAt:      hourLater,
					DeletedAt:      lo.ToPtr(now.Add(-30 * time.Minute)),
					DeletedComment: lo.ToPtr("bar"),
				},
			},

			expectErr: dao.ErrShortCodeNotFound,
		},
	}

	repository := dao.NewDeleteShortCodeRepository()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			postgres.RunTransactionalTest(t, testutils.TestDBConfig, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				key, err := repository.DeleteShortCode(ctx, testCase.data)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, key)
			})
		})
	}
}
