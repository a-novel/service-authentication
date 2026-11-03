package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/models/migrations"
)

func TestPgCredentialsUpdateLocationConsent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.CredentialsUpdateLocationConsentRequest

		expectConsent bool
		expectErr     error
	}{
		{
			name: "Success/On",

			request: &dao.CredentialsUpdateLocationConsentRequest{ID: sessionOwnerID, LocationConsent: true},

			expectConsent: true,
		},
		{
			name: "Success/Off",

			request: &dao.CredentialsUpdateLocationConsentRequest{ID: sessionOwnerID},
		},
		{
			name: "Error/NotFound",

			request: &dao.CredentialsUpdateLocationConsentRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), LocationConsent: true,
			},

			expectErr: dao.ErrCredentialsUpdateLocationConsentNotFound,
		},
	}

	daoUpdate := dao.NewPgCredentialsUpdateLocationConsent()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				sessionFixtures{}.insert(ctx, t)

				credentials, err := daoUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				if testCase.expectErr == nil {
					require.Equal(t, testCase.request.ID, credentials.ID)
					require.Equal(t, testCase.expectConsent, credentials.LocationConsent)
				}
			})
		})
	}
}
