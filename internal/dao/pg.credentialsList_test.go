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

// The listing orders by created_at DESC, id DESC. The fixtures order created_at and updated_at
// differently, so these assertions fail on an updated_at ordering.
func TestCredentialsList(t *testing.T) {
	t.Parallel()

	// created_at descends 1 > 2 > 3, and updated_at ascends.
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
		{
			name:     "Success/ExactEmails",
			fixtures: []*dao.Credentials{cred1, cred2, cred3},
			request:  &dao.CredentialsListRequest{Emails: []string{cred1.Email, "User2@email.com", "missing@email.com"}},
			expect:   []*dao.Credentials{cred1},
		},
		{
			name:     "Success/EmailsAndRoles",
			fixtures: []*dao.Credentials{cred1, cred2, cred3},
			request:  &dao.CredentialsListRequest{Emails: []string{cred1.Email, cred2.Email}, Roles: []string{"auth:admin"}},
			expect:   []*dao.Credentials{cred2},
		},
		{
			name:     "Success/NoMatchingEmails",
			fixtures: []*dao.Credentials{cred1},
			request:  &dao.CredentialsListRequest{Emails: []string{"missing@email.com"}},
			expect:   []*dao.Credentials{},
		},
	}

	listDAO := dao.NewCredentialsList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
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

// Pages order on the immutable created_at, with id breaking ties. Each case reads the whole set in
// two pages and asserts their union holds every inserted row exactly once.
func TestCredentialsListPaginationIsStable(t *testing.T) {
	t.Parallel()

	listDAO := dao.NewCredentialsList()

	t.Run("Success/SameSecondTies", func(t *testing.T) {
		t.Parallel()

		postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			// Four rows share one created_at, a tie timestamp(0) truncation makes common. The id
			// tiebreaker keeps their order stable across queries.
			ts := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
			fixtures := make([]*dao.Credentials, 0, 4)

			for i := 1; i <= 4; i++ {
				fixture := &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-00000000000" + string(rune('0'+i))),
					Email:     "user" + string(rune('0'+i)) + "@email.com",
					Role:      "auth:user",
					CreatedAt: ts,
					UpdatedAt: ts,
				}
				fixtures = append(fixtures, fixture)
			}

			_, err = db.NewInsert().Model(&fixtures).Exec(ctx)
			require.NoError(t, err)

			page1, err := listDAO.Exec(ctx, &dao.CredentialsListRequest{Limit: 2, Offset: 0})
			require.NoError(t, err)

			page2, err := listDAO.Exec(ctx, &dao.CredentialsListRequest{Limit: 2, Offset: 2})
			require.NoError(t, err)

			seen := countIDs(append(append([]*dao.Credentials{}, page1...), page2...))
			require.Len(t, seen, 4, "the two pages must union to all four rows")

			for id, count := range seen {
				require.Equal(t, 1, count, "row %s appeared %d times across the pages", id, count)
			}
		})
	})

	t.Run("Success/RowUpdatedBetweenPages", func(t *testing.T) {
		t.Parallel()

		postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			db, err := postgres.GetContext(ctx)
			require.NoError(t, err)

			// Distinct created_at values isolate the update: the touched row keeps its place.
			fixtures := make([]*dao.Credentials, 0, 4)

			for i := 1; i <= 4; i++ {
				fixture := &dao.Credentials{
					ID:        uuid.MustParse("00000000-0000-0000-0000-00000000000" + string(rune('0'+i))),
					Email:     "user" + string(rune('0'+i)) + "@email.com",
					Role:      "auth:user",
					CreatedAt: time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC),
				}
				fixtures = append(fixtures, fixture)
			}

			_, err = db.NewInsert().Model(&fixtures).Exec(ctx)
			require.NoError(t, err)

			page1, err := listDAO.Exec(ctx, &dao.CredentialsListRequest{Limit: 2, Offset: 0})
			require.NoError(t, err)

			// Give the oldest row the newest updated_at, which leaves its page unchanged.
			_, err = db.NewUpdate().
				Model((*dao.Credentials)(nil)).
				Set("updated_at = ?", time.Date(2021, 1, 31, 0, 0, 0, 0, time.UTC)).
				Where("id = ?", fixtures[0].ID).
				Exec(ctx)
			require.NoError(t, err)

			page2, err := listDAO.Exec(ctx, &dao.CredentialsListRequest{Limit: 2, Offset: 2})
			require.NoError(t, err)

			seen := countIDs(append(append([]*dao.Credentials{}, page1...), page2...))
			require.Len(t, seen, 4, "the mutation must not drop a row from the union")

			for id, count := range seen {
				require.Equal(t, 1, count, "row %s appeared %d times across the pages", id, count)
			}
		})
	})
}

// countIDs counts how many times each ID appears among credentials.
func countIDs(credentials []*dao.Credentials) map[uuid.UUID]int {
	seen := map[uuid.UUID]int{}
	for _, entry := range credentials {
		seen[entry.ID]++
	}

	return seen
}
