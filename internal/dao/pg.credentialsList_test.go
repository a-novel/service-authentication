package dao_test

import (
	"context"
	"fmt"
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
func TestPgCredentialsList(t *testing.T) {
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

	// These rows share one created_at, so only their id orders them.
	tied := make([]*dao.Credentials, 4)
	for i := range tied {
		tied[i] = &dao.Credentials{
			ID:        uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-00000000000%d", i+1)),
			Email:     fmt.Sprintf("tied%d@email.com", i+1),
			Role:      "auth:user",
			CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		}
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
			// cred3 holds the newest updated_at and stays last, so an update never moves a row
			// across pages.
			name: "Success/Offset",

			fixtures: []*dao.Credentials{cred1, cred2, cred3},
			request:  &dao.CredentialsListRequest{Limit: 2, Offset: 1},
			expect:   []*dao.Credentials{cred2, cred3},
		},
		{
			name: "Success/TiedFirstPage",

			fixtures: tied,
			request:  &dao.CredentialsListRequest{Limit: 2},
			expect:   []*dao.Credentials{tied[3], tied[2]}, // id DESC
		},
		{
			// With the first page, it holds every tied row exactly once.
			name: "Success/TiedSecondPage",

			fixtures: tied,
			request:  &dao.CredentialsListRequest{Limit: 2, Offset: 2},
			expect:   []*dao.Credentials{tied[1], tied[0]},
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

	listDAO := dao.NewPgCredentialsList()

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
