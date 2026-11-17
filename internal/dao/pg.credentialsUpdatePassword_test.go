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

func TestPgCredentialsUpdatePassword(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		fixtures []*dao.Credentials

		request *dao.CredentialsUpdatePasswordRequest

		expect    *dao.Credentials
		expectErr error
	}{
		{
			name: "Success",

			fixtures: []*dao.Credentials{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					Role:      "auth:user",
				},
			},

			request: &dao.CredentialsUpdatePasswordRequest{
				ID:       uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Password: "new-password-hashed",
				Now:      time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Email:     "user@provider.com",
				Password:  "new-password-hashed",
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				Role:      "auth:user",
			},
		},
		{
			name: "Success/RemovePassword",

			fixtures: []*dao.Credentials{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					Role:      "auth:user",
				},
			},

			request: &dao.CredentialsUpdatePasswordRequest{
				ID:  uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Now: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Email:     "user@provider.com",
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				Role:      "auth:user",
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.CredentialsUpdatePasswordRequest{
				ID:       uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Password: "new-password-hashed",
				Now:      time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expectErr: dao.ErrCredentialsNotFound,
		},
	}

	daoUpdatePassword := dao.NewPgCredentialsUpdatePassword()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, testCase.fixtures)

				credentials, err := daoUpdatePassword.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, credentials)
			})
		})
	}
}
