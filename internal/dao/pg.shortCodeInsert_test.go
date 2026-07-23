package dao_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestShortCodeInsert(t *testing.T) {
	t.Parallel()

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

	dao := dao.NewShortCodeInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				if len(testCase.fixtures) > 0 {
					_, err = db.NewInsert().Model(&testCase.fixtures).Exec(ctx)
					require.NoError(t, err)
				}

				credentials, err := dao.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, credentials)
			})
		})
	}
}

// TestShortCodeInsertConcurrentOverride races two Override inserts for one
// (target, usage) pair against a single active code they both contend on. The partial
// unique index guarantees at most one active row survives, so on every interleaving
// the loser must take the ordinary already-exists path — never a raw SQLSTATE 40001
// serialization failure, which a repeatable-read transaction would surface here as a
// 500. Several rounds run so the two transactions reliably overlap at least once; a
// repeatable-read implementation fails the round where they do.
func TestShortCodeInsertConcurrentOverride(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	hourAgo := time.Now().Add(-time.Hour).UTC().Round(time.Second)
	hourLater := time.Now().Add(time.Hour).UTC().Round(time.Second)

	shortCodeDAO := dao.NewShortCodeInsert()

	postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
		t.Helper()

		db, err := postgres.GetContext(ctx)
		require.NoError(t, err)

		// Seed the single active code the first round's overrides contend on. Every
		// later round contends on the previous round's surviving row.
		seed := []*dao.ShortCode{
			{
				ID:        uuid.New(),
				Code:      "seed-code",
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("seed-data"),
				CreatedAt: hourAgo,
				ExpiresAt: hourLater,
			},
		}
		_, err = db.NewInsert().Model(&seed).Exec(ctx)
		require.NoError(t, err)

		const rounds = 10

		for round := range rounds {
			// Two overrides for the same pair, released together so their transactions
			// overlap and genuinely contend on the current active row's override UPDATE.
			requests := []*dao.ShortCodeInsertRequest{
				{
					ID: uuid.New(), Code: "override-a", Usage: "test-usage", Target: "test-target",
					Data: []byte("data-a"), Now: now, ExpiresAt: hourLater, Override: true,
				},
				{
					ID: uuid.New(), Code: "override-b", Usage: "test-usage", Target: "test-target",
					Data: []byte("data-b"), Now: now, ExpiresAt: hourLater, Override: true,
				},
			}

			start := make(chan struct{})
			errs := make([]error, len(requests))

			var wg sync.WaitGroup
			for i, request := range requests {
				wg.Add(1)

				go func() {
					defer wg.Done()

					<-start

					_, errs[i] = shortCodeDAO.Exec(ctx, request)
				}()
			}

			close(start)
			wg.Wait()

			// The loser never sees a raw serialization failure — only the mapped
			// already-exists sentinel — and at least one override always lands.
			successes := 0

			for _, err := range errs {
				if err == nil {
					successes++

					continue
				}

				require.ErrorIs(t, err, dao.ErrShortCodeInsertAlreadyExists)
			}

			require.GreaterOrEqualf(t, successes, 1, "round %d: at least one override must succeed", round)

			// The invariant the unique index protects: exactly one active row remains.
			activeCount, err := db.NewSelect().
				Model((*dao.ShortCode)(nil)).
				Where("target = ?", "test-target").
				Where("usage = ?", "test-usage").
				Where("deleted_at IS NULL").
				Count(ctx)
			require.NoError(t, err)
			require.Equalf(t, 1, activeCount, "round %d: exactly one active short code must remain", round)
		}
	})
}
