package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestCredentialsUpdateEmail(t *testing.T) {
	testCases := []struct {
		name string

		fixtures []*dao.Credentials

		request *dao.CredentialsUpdateEmailRequest

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
					Role:      "test_role_user",
				},
			},

			request: &dao.CredentialsUpdateEmailRequest{
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Email: "new-user@provider.com",
				Now:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expect: &dao.Credentials{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Email:     "new-user@provider.com",
				Password:  "password-2-hashed",
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				Role:      "test_role_user",
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.CredentialsUpdateEmailRequest{
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Email: "new-user@provider.com",
				Now:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expectErr: dao.ErrCredentialsUpdateEmailNotFound,
		},
		{
			name: "Error/EmailTaken",

			fixtures: []*dao.Credentials{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "new-user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC),
					Role:      "test_role_user",
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					Role:      "test_role_user",
				},
			},

			request: &dao.CredentialsUpdateEmailRequest{
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				Email: "new-user@provider.com",
				Now:   time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},

			expectErr: dao.ErrCredentialsUpdateEmailAlreadyExists,
		},
	}

	repository := dao.NewCredentialsUpdateEmail()

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
