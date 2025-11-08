package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/dao"
)

func TestShortCodeInsert(t *testing.T) {
	now := time.Now().UTC().Round(time.Second)
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	testCases := []struct {
		name string

		fixtures []*dao.ShortCode

		request *dao.ShortCodeInsertRequest

		expect    *dao.ShortCode
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ShortCodeInsertRequest{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				Now:       now,
				ExpiresAt: hourLater,
			},

			expect: &dao.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: now,
				ExpiresAt: hourLater,
			},
		},
		{
			name: "Success/NoData",

			request: &dao.ShortCodeInsertRequest{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Now:       now,
				ExpiresAt: hourLater,
			},

			expect: &dao.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				CreatedAt: now,
				ExpiresAt: hourLater,
			},
		},
		{
			name: "SameTargetDifferentUsage",

			request: &dao.ShortCodeInsertRequest{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				Now:       now,
				ExpiresAt: hourLater,
			},

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Code:      "test-code",
					Usage:     "test-usage-2",
					Target:    "test-target",
					Data:      []byte("test-data-2"),
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
			},

			expect: &dao.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: now,
				ExpiresAt: hourLater,
			},
		},
		{
			name: "SameUsageDifferentTarget",

			request: &dao.ShortCodeInsertRequest{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				Now:       now,
				ExpiresAt: hourLater,
			},

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Code:      "test-code",
					Usage:     "test-usage",
					Target:    "test-target-2",
					Data:      []byte("test-data-2"),
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
			},

			expect: &dao.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: now,
				ExpiresAt: hourLater,
			},
		},
		{
			name: "Error/AlreadyExists",

			request: &dao.ShortCodeInsertRequest{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				Now:       now,
				ExpiresAt: hourLater,
			},

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Code:      "test-code-2",
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data-2"),
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
			},

			expectErr: dao.ErrShortCodeInsertAlreadyExists,
		},
		{
			name: "AlreadyExists/Override",

			request: &dao.ShortCodeInsertRequest{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				Now:       now,
				ExpiresAt: hourLater,
				Override:  true,
			},

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Code:      "test-code",
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data-2"),
					CreatedAt: hourAgo,
					ExpiresAt: hourLater,
				},
			},

			expect: &dao.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: now,
				ExpiresAt: hourLater,
			},
		},
		{
			name: "AlreadyExists/AlreadyExpired",

			request: &dao.ShortCodeInsertRequest{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				Now:       now,
				ExpiresAt: hourLater,
			},

			fixtures: []*dao.ShortCode{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Code:      "test-code",
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data-2"),
					CreatedAt: hourAgo,
					ExpiresAt: hourAgo,
				},
			},

			expect: &dao.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: now,
				ExpiresAt: hourLater,
			},
		},
		{
			name: "AlreadyExistsAlreadyDeleted",

			request: &dao.ShortCodeInsertRequest{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				Now:       now,
				ExpiresAt: hourLater,
			},

			fixtures: []*dao.ShortCode{
				{
					ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Code:           "test-code",
					Usage:          "test-usage",
					Target:         "test-target",
					Data:           []byte("test-data-2"),
					CreatedAt:      hourAgo,
					ExpiresAt:      hourLater,
					DeletedAt:      &hourAgo,
					DeletedComment: lo.ToPtr("test-comment"),
				},
			},

			expect: &dao.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Code:      "test-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: now,
				ExpiresAt: hourLater,
			},
		},
	}

	repository := dao.NewShortCodeInsert()

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

				credentials, err := repository.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, credentials)
			})
		})
	}
}
