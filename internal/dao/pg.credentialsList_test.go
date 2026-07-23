package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

// The listing orders by created_at DESC, id DESC. The fixtures below deliberately put
// created_at and updated_at in different orders, so a revert to the old updated_at sort
// produces a different slice and fails these assertions rather than passing by
// coincidence.
func TestCredentialsList(t *testing.T) {
	t.Parallel()

	// created_at descends 1 > 2 > 3; updated_at ascends, so ordering on it would reverse
	// the result.
	cred1 := &dao.Credentials{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email:     "user1@email.com",
		Role:      "auth:user",
		CreatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	cred2 := &dao.Credentials{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Email:     "user2@email.com",
		Role:      "auth:admin",
		CreatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	cred3 := &dao.Credentials{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		Email:     "user3@email.com",
		Role:      "auth:user",
		CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
	}

	testCases := []struct {
		name string

		fixtures []*dao.Credentials

		request *dao.CredentialsListRequest

		expect    []*dao.Credentials
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.Credentials{cred1, cred2, cred3},
			request:  &dao.CredentialsListRequest{},
			expect:   []*dao.Credentials{cred1, cred2, cred3}, // created_at DESC
		},
		{
			name: "Success/Limit",

			fixtures: []*dao.Credentials{cred1, cred2, cred3},
			request:  &dao.CredentialsListRequest{Limit: 2},
			expect:   []*dao.Credentials{cred1, cred2},
		},
		{
			name: "Success/Offset",

			fixtures: []*dao.Credentials{cred1, cred2, cred3},
			request:  &dao.CredentialsListRequest{Limit: 2, Offset: 1},
			expect:   []*dao.Credentials{cred2, cred3},
		},
		{
			name: "Success/RoleFilter",

			fixtures: []*dao.Credentials{cred1, cred2, cred3},
			request:  &dao.CredentialsListRequest{Roles: []string{"auth:user"}},
			expect:   []*dao.Credentials{cred1, cred3}, // admin cred2 excluded
		},
	}

	listDAO := dao.NewCredentialsList()

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

				credentials, err := listDAO.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, credentials)
			})
		})
	}
}

// The two ways offset pagination over updated_at lost rows, both fixed by ordering on the
// immutable created_at with id as the tiebreaker. Each pages through the whole set in
// two reads and asserts the union is exactly what was inserted — no row skipped, none
// repeated.
func TestCredentialsListPaginationIsStable(t *testing.T) {
	t.Parallel()

	listDAO := dao.NewCredentialsList()

	ids := func(creds []*dao.Credentials) map[uuid.UUID]int {
		seen := map[uuid.UUID]int{}
		for _, c := range creds {
			seen[c.ID]++
		}

		return seen
	}

	t.Run("same-second ties page without loss", func(t *testing.T) {
		t.Parallel()

		postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			// Four rows sharing one created_at — the tie timestamp(0) truncation makes
			// common. Without the id tiebreaker the planner could order the group
			// differently per query, so a boundary row is skipped or repeated.
			ts := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
			fixtures := make([]*dao.Credentials, 0, 4)

			for i := 1; i <= 4; i++ {
				fixtures = append(fixtures, &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-00000000000" + string(rune('0'+i))),
					Email:     "user" + string(rune('0'+i)) + "@email.com",
					Role:      "auth:user",
					CreatedAt: ts,
					UpdatedAt: ts,
				})
			}

			_, err = db.NewInsert().Model(&fixtures).Exec(ctx)
			require.NoError(t, err)

			page1, err := listDAO.Exec(ctx, &dao.CredentialsListRequest{Limit: 2, Offset: 0})
			require.NoError(t, err)

			page2, err := listDAO.Exec(ctx, &dao.CredentialsListRequest{Limit: 2, Offset: 2})
			require.NoError(t, err)

			seen := ids(append(append([]*dao.Credentials{}, page1...), page2...))
			require.Len(t, seen, 4, "the two pages must union to all four rows")

			for id, count := range seen {
				require.Equal(t, 1, count, "row %s appeared %d times across the pages", id, count)
			}
		})
	})

	t.Run("a row touched between pages is neither skipped nor repeated", func(t *testing.T) {
		t.Parallel()

		postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			// Distinct created_at, so this is purely the mutation problem: ordering on
			// updated_at would move the touched row across the page boundary.
			fixtures := make([]*dao.Credentials, 0, 4)

			for i := 1; i <= 4; i++ {
				fixtures = append(fixtures, &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-00000000000" + string(rune('0'+i))),
					Email:     "user" + string(rune('0'+i)) + "@email.com",
					Role:      "auth:user",
					CreatedAt: time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC),
				})
			}

			_, err = db.NewInsert().Model(&fixtures).Exec(ctx)
			require.NoError(t, err)

			page1, err := listDAO.Exec(ctx, &dao.CredentialsListRequest{Limit: 2, Offset: 0})
			require.NoError(t, err)

			// The touch the old ordering was vulnerable to: bump the oldest-created row
			// to the newest updated_at, which would drag it to the head of an
			// updated_at ordering and across the page boundary.
			_, err = db.NewUpdate().
				Model((*dao.Credentials)(nil)).
				Set("updated_at = ?", time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC)).
				Where("id = ?", fixtures[0].ID).
				Exec(ctx)
			require.NoError(t, err)

			page2, err := listDAO.Exec(ctx, &dao.CredentialsListRequest{Limit: 2, Offset: 2})
			require.NoError(t, err)

			seen := ids(append(append([]*dao.Credentials{}, page1...), page2...))
			require.Len(t, seen, 4, "the mutation must not drop a row from the union")

			for id, count := range seen {
				require.Equal(t, 1, count, "row %s appeared %d times across the pages", id, count)
			}
		})
	})
}
